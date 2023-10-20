package dag_test

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

// Note: you can find explanations about the underlying data model here:
// https://github.com/MichaelMure/git-bug/blob/master/doc/model.md

// This file explains how to define a replicated data structure, stored in and using git as a medium for
// synchronisation. To do this, we'll use the entity/dag package, which will do all the complex handling.
//
// The example we'll use here is a small shared configuration with two fields. One of them is special as
// it also defines who is allowed to change said configuration.
// Note: this example is voluntarily a bit complex with operation linking to identities and logic rules,
// to show that how something more complex than a toy would look like. That said, it's still a simplified
// example: in git-bug for example, more layers are added for caching, memory handling and to provide an
// easier to use API.
//
// Let's start by defining the document/structure we are going to share:

// Snapshot is the compiled view of a ProjectConfig
type Snapshot struct {
	// Administrator is the set of users with the higher level of access
	Administrator map[identity.Interface]struct{}
	// SignatureRequired indicate that all git commit need to be signed
	SignatureRequired bool
}

// HasAdministrator returns true if the given identity is included in the administrator.
func (snap *Snapshot) HasAdministrator(i identity.Interface) bool {
	for admin, _ := range snap.Administrator {
		if admin.Id() == i.Id() {
			return true
		}
	}
	return false
}

// Now, we will not edit this configuration directly. Instead, we are going to apply "operations" on it.
// Those are the ones that will be stored and shared. Doing things that way allow merging concurrent editing
// and deal with conflict.
//
// Here, we will define three operations:
// - SetSignatureRequired is a simple operation that set or unset the SignatureRequired boolean
// - AddAdministrator is more complex and add a new administrator in the Administrator set
// - RemoveAdministrator is the counterpart the remove administrators
//
// Note: there is some amount of boilerplate for operations. In a real project, some of that can be
// factorized and simplified.

// Operation is the operation interface acting on Snapshot
type Operation interface {
	dag.Operation

	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot *Snapshot)
}

const (
	_ entity.OperationType = iota
	SetSignatureRequiredOp
	AddAdministratorOp
	RemoveAdministratorOp
)

// SetSignatureRequired is an operation to set/unset if git signature are required.
type SetSignatureRequired struct {
	dag.OpBase
	Value bool `json:"value"`
}

func NewSetSignatureRequired(author identity.Interface, value bool) *SetSignatureRequired {
	return &SetSignatureRequired{
		OpBase: dag.NewOpBase(SetSignatureRequiredOp, author, time.Now().Unix()),
		Value:  value,
	}
}

func (ssr *SetSignatureRequired) Id() entity.Id {
	// the Id of the operation is the hash of the serialized data.
	return dag.IdOperation(ssr, &ssr.OpBase)
}

func (ssr *SetSignatureRequired) Validate() error {
	return ssr.OpBase.Validate(ssr, SetSignatureRequiredOp)
}

// Apply is the function that makes changes on the snapshot
func (ssr *SetSignatureRequired) Apply(snapshot *Snapshot) {
	// check that we are allowed to change the config
	if _, ok := snapshot.Administrator[ssr.Author()]; !ok {
		return
	}
	snapshot.SignatureRequired = ssr.Value
}

// AddAdministrator is an operation to add a new administrator in the set
type AddAdministrator struct {
	dag.OpBase
	ToAdd []identity.Interface `json:"to_add"`
}

func NewAddAdministratorOp(author identity.Interface, toAdd ...identity.Interface) *AddAdministrator {
	return &AddAdministrator{
		OpBase: dag.NewOpBase(AddAdministratorOp, author, time.Now().Unix()),
		ToAdd:  toAdd,
	}
}

func (aa *AddAdministrator) Id() entity.Id {
	// the Id of the operation is the hash of the serialized data.
	return dag.IdOperation(aa, &aa.OpBase)
}

func (aa *AddAdministrator) Validate() error {
	// Let's enforce an arbitrary rule
	if len(aa.ToAdd) == 0 {
		return fmt.Errorf("nothing to add")
	}
	return aa.OpBase.Validate(aa, AddAdministratorOp)
}

// Apply is the function that makes changes on the snapshot
func (aa *AddAdministrator) Apply(snapshot *Snapshot) {
	// check that we are allowed to change the config ... or if there is no admin yet
	if !snapshot.HasAdministrator(aa.Author()) && len(snapshot.Administrator) != 0 {
		return
	}
	for _, toAdd := range aa.ToAdd {
		snapshot.Administrator[toAdd] = struct{}{}
	}
}

// RemoveAdministrator is an operation to remove an administrator from the set
type RemoveAdministrator struct {
	dag.OpBase
	ToRemove []identity.Interface `json:"to_remove"`
}

func NewRemoveAdministratorOp(author identity.Interface, toRemove ...identity.Interface) *RemoveAdministrator {
	return &RemoveAdministrator{
		OpBase:   dag.NewOpBase(RemoveAdministratorOp, author, time.Now().Unix()),
		ToRemove: toRemove,
	}
}

func (ra *RemoveAdministrator) Id() entity.Id {
	// the Id of the operation is the hash of the serialized data.
	return dag.IdOperation(ra, &ra.OpBase)
}

func (ra *RemoveAdministrator) Validate() error {
	// Let's enforce some rules. If we return an error, this operation will be
	// considered invalid and will not be included in our data.
	if len(ra.ToRemove) == 0 {
		return fmt.Errorf("nothing to remove")
	}
	return ra.OpBase.Validate(ra, RemoveAdministratorOp)
}

// Apply is the function that makes changes on the snapshot
func (ra *RemoveAdministrator) Apply(snapshot *Snapshot) {
	// check if we are allowed to make changes
	if !snapshot.HasAdministrator(ra.Author()) {
		return
	}
	// special rule: we can't end up with no administrator
	stillSome := false
	for admin, _ := range snapshot.Administrator {
		if admin != ra.Author() {
			stillSome = true
			break
		}
	}
	if !stillSome {
		return
	}
	// apply
	for _, toRemove := range ra.ToRemove {
		delete(snapshot.Administrator, toRemove)
	}
}

// Now, let's create the main object (the entity) we are going to manipulate: ProjectConfig.
// This object wrap a dag.Entity, which makes it inherit some methods and provide all the complex
// DAG handling. Additionally, ProjectConfig is the place where we can add functions specific for that type.

type ProjectConfig struct {
	// this is really all we need
	*dag.Entity
}

func NewProjectConfig() *ProjectConfig {
	return wrapper(dag.New(def))
}

func wrapper(e *dag.Entity) *ProjectConfig {
	return &ProjectConfig{Entity: e}
}

// a Definition describes a few properties of the Entity, a sort of configuration to manipulate the
// DAG of operations
var def = dag.Definition{
	Typename:             "project config",
	Namespace:            "conf",
	OperationUnmarshaler: operationUnmarshaler,
	FormatVersion:        1,
}

// operationUnmarshaler is a function doing the de-serialization of the JSON data into our own
// concrete Operations. If needed, we can use the resolver to connect to other entities.
func operationUnmarshaler(raw json.RawMessage, resolvers entity.Resolvers) (dag.Operation, error) {
	var t struct {
		OperationType entity.OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op dag.Operation

	switch t.OperationType {
	case AddAdministratorOp:
		op = &AddAdministrator{}
	case RemoveAdministratorOp:
		op = &RemoveAdministrator{}
	case SetSignatureRequiredOp:
		op = &SetSignatureRequired{}
	default:
		panic(fmt.Sprintf("unknown operation type %v", t.OperationType))
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	switch op := op.(type) {
	case *AddAdministrator:
		// We need to resolve identities
		for i, stub := range op.ToAdd {
			iden, err := entity.Resolve[identity.Interface](resolvers, stub.Id())
			if err != nil {
				return nil, err
			}
			op.ToAdd[i] = iden
		}
	case *RemoveAdministrator:
		// We need to resolve identities
		for i, stub := range op.ToRemove {
			iden, err := entity.Resolve[identity.Interface](resolvers, stub.Id())
			if err != nil {
				return nil, err
			}
			op.ToRemove[i] = iden
		}
	}

	return op, nil
}

// Compile compute a view of the final state. This is what we would use to display the state
// in a user interface.
func (pc ProjectConfig) Compile() *Snapshot {
	// Note: this would benefit from caching, but it's a simple example
	snap := &Snapshot{
		// default value
		Administrator:     make(map[identity.Interface]struct{}),
		SignatureRequired: false,
	}
	for _, op := range pc.Operations() {
		op.(Operation).Apply(snap)
	}
	return snap
}

// Read is a helper to load a ProjectConfig from a Repository
func Read(repo repository.ClockedRepo, id entity.Id) (*ProjectConfig, error) {
	return dag.Read(def, wrapper, repo, simpleResolvers(repo), id)
}

func simpleResolvers(repo repository.ClockedRepo) entity.Resolvers {
	// resolvers can look a bit complex or out of place here, but it's an important concept
	// to allow caching and flexibility when constructing the final app.
	return entity.Resolvers{
		&identity.Identity{}: identity.NewSimpleResolver(repo),
	}
}

func Example_entity() {
	const gitBugNamespace = "git-bug"
	// Note: this example ignore errors for readability
	// Note: variable names get a little confusing as we are simulating both side in the same function

	// Let's start by defining two git repository and connecting them as remote
	repoRenePath, _ := os.MkdirTemp("", "")
	repoIsaacPath, _ := os.MkdirTemp("", "")
	repoRene, _ := repository.InitGoGitRepo(repoRenePath, gitBugNamespace)
	defer repoRene.Close()
	repoIsaac, _ := repository.InitGoGitRepo(repoIsaacPath, gitBugNamespace)
	defer repoIsaac.Close()
	_ = repoRene.AddRemote("origin", repoIsaacPath)
	_ = repoIsaac.AddRemote("origin", repoRenePath)

	// Now we need identities and to propagate them
	rene, _ := identity.NewIdentity(repoRene, "René Descartes", "rene@descartes.fr")
	isaac, _ := identity.NewIdentity(repoRene, "Isaac Newton", "isaac@newton.uk")
	_ = rene.Commit(repoRene)
	_ = isaac.Commit(repoRene)
	_ = identity.Pull(repoIsaac, "origin")

	// create a new entity
	confRene := NewProjectConfig()

	// add some operations
	confRene.Append(NewAddAdministratorOp(rene, rene))
	confRene.Append(NewAddAdministratorOp(rene, isaac))
	confRene.Append(NewSetSignatureRequired(rene, true))

	// Rene commits on its own repo
	_ = confRene.Commit(repoRene)

	// Isaac pull and read the config
	_ = dag.Pull(def, wrapper, repoIsaac, simpleResolvers(repoIsaac), "origin", isaac)
	confIsaac, _ := Read(repoIsaac, confRene.Id())

	// Compile gives the current state of the config
	snapshot := confIsaac.Compile()
	for admin, _ := range snapshot.Administrator {
		fmt.Println(admin.DisplayName())
	}

	// Isaac add more operations
	confIsaac.Append(NewSetSignatureRequired(isaac, false))
	reneFromIsaacRepo, _ := identity.ReadLocal(repoIsaac, rene.Id())
	confIsaac.Append(NewRemoveAdministratorOp(isaac, reneFromIsaacRepo))
	_ = confIsaac.Commit(repoIsaac)
}
