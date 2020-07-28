// Package core contains the target-agnostic code to define and run a bridge
package core

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

var ErrImportNotSupported = errors.New("import is not supported")
var ErrExportNotSupported = errors.New("export is not supported")

const (
	ConfigKeyTarget = "target"

	MetaKeyOrigin = "origin"

	bridgeConfigKeyPrefix = "git-bug.bridge"
)

var bridgeImpl map[string]reflect.Type
var bridgeValidParams map[string][]string
var bridgeLoginMetaKey map[string]string

// Bridge is a wrapper around a BridgeImpl that will bind low-level
// implementation with utility code to provide high-level functions.
type Bridge struct {
	Name           string
	repo           *cache.RepoCache
	impl           BridgeImpl
	importer       Importer
	exporter       Exporter
	conf           Configuration
	initImportDone bool
	initExportDone bool
}

// Register will register a new BridgeImpl
func Register(impl BridgeImpl) {
	if bridgeImpl == nil {
		bridgeImpl = make(map[string]reflect.Type)
	}
	if bridgeLoginMetaKey == nil {
		bridgeLoginMetaKey = make(map[string]string)
	}
	if bridgeValidParams == nil {
		bridgeValidParams = make(map[string][]string)
	}
	bridgeImpl[impl.Target()] = reflect.TypeOf(impl).Elem()
	bridgeLoginMetaKey[impl.Target()] = impl.LoginMetaKey()

	paramMap := impl.ValidParams()
	params := make([]string, len(paramMap))

	i := 0
	for k := range paramMap {
		params[i] = k
		i++
	}
	bridgeValidParams[impl.Target()] = params
}

// Targets return all known bridge implementation target
func Targets() []string {
	var result []string

	for key := range bridgeImpl {
		result = append(result, key)
	}

	sort.Strings(result)

	return result
}

// ValidParams returns the valid input parameters for configuring a target bridge
func ValidParams(target string) ([]string, error) {
	validParams, ok := bridgeValidParams[target]
	if !ok {
		return nil, fmt.Errorf("unknown bridge target %v", target)
	}

	sort.Strings(validParams)
	return validParams, nil
}

// TargetExist return true if the given target has a bridge implementation
func TargetExist(target string) bool {
	_, ok := bridgeImpl[target]
	return ok
}

// LoginMetaKey return the metadata key used to store the remote bug-tracker login
// on the user identity. The corresponding value is used to match identities and
// credentials.
func LoginMetaKey(target string) (string, error) {
	metaKey, ok := bridgeLoginMetaKey[target]
	if !ok {
		return "", fmt.Errorf("unknown bridge target %v", target)
	}

	return metaKey, nil
}

// Instantiate a new Bridge for a repo, from the given target and name
func NewBridge(repo *cache.RepoCache, target string, name string) (*Bridge, error) {
	implType, ok := bridgeImpl[target]
	if !ok {
		return nil, fmt.Errorf("unknown bridge target %v", target)
	}

	impl := reflect.New(implType).Interface().(BridgeImpl)

	bridge := &Bridge{
		Name: name,
		repo: repo,
		impl: impl,
	}

	return bridge, nil
}

// LoadBridge instantiate a new bridge from a repo configuration
func LoadBridge(repo *cache.RepoCache, name string) (*Bridge, error) {
	conf, err := loadConfig(repo, name)
	if err != nil {
		return nil, err
	}

	target := conf[ConfigKeyTarget]
	bridge, err := NewBridge(repo, target, name)
	if err != nil {
		return nil, err
	}

	err = bridge.impl.ValidateConfig(conf)
	if err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	// will avoid reloading configuration before an export or import call
	bridge.conf = conf
	return bridge, nil
}

// Attempt to retrieve a default bridge for the given repo. If zero or multiple
// bridge exist, it fails.
func DefaultBridge(repo *cache.RepoCache) (*Bridge, error) {
	bridges, err := ConfiguredBridges(repo)
	if err != nil {
		return nil, err
	}

	if len(bridges) == 0 {
		return nil, fmt.Errorf("no configured bridge")
	}

	if len(bridges) > 1 {
		return nil, fmt.Errorf("multiple bridge are configured, you need to select one explicitely")
	}

	return LoadBridge(repo, bridges[0])
}

// ConfiguredBridges return the list of bridge that are configured for the given
// repo
func ConfiguredBridges(repo repository.RepoConfig) ([]string, error) {
	configs, err := repo.LocalConfig().ReadAll(bridgeConfigKeyPrefix + ".")
	if err != nil {
		return nil, errors.Wrap(err, "can't read configured bridges")
	}

	re := regexp.MustCompile(bridgeConfigKeyPrefix + `.([^.]+)`)

	set := make(map[string]interface{})

	for key := range configs {
		res := re.FindStringSubmatch(key)

		if res == nil {
			continue
		}

		set[res[1]] = nil
	}

	result := make([]string, len(set))

	i := 0
	for key := range set {
		result[i] = key
		i++
	}

	return result, nil
}

// Check if a bridge exist
func BridgeExist(repo repository.RepoConfig, name string) bool {
	keyPrefix := fmt.Sprintf("git-bug.bridge.%s.", name)

	conf, err := repo.LocalConfig().ReadAll(keyPrefix)

	return err == nil && len(conf) > 0
}

// Remove a configured bridge
func RemoveBridge(repo repository.RepoConfig, name string) error {
	re := regexp.MustCompile(`^[a-zA-Z0-9]+`)

	if !re.MatchString(name) {
		return fmt.Errorf("bad bridge fullname: %s", name)
	}

	keyPrefix := fmt.Sprintf("git-bug.bridge.%s", name)
	return repo.LocalConfig().RemoveAll(keyPrefix)
}

// Configure run the target specific configuration process
func (b *Bridge) Configure(params BridgeParams) error {
	validateParams(params, b.impl)

	conf, err := b.impl.Configure(b.repo, params)
	if err != nil {
		return err
	}

	err = b.impl.ValidateConfig(conf)
	if err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	b.conf = conf
	return b.storeConfig(conf)
}

func validateParams(params BridgeParams, impl BridgeImpl) {
	validParams := impl.ValidParams()

	paramsValue := reflect.ValueOf(params)
	paramsType := paramsValue.Type()

	for i := 0; i < paramsValue.NumField(); i++ {
		name := paramsType.Field(i).Name
		val := paramsValue.Field(i).Interface().(string)
		_, valid := validParams[name]
		if val != "" && !valid {
			_, _ = fmt.Fprintln(os.Stderr, params.fieldWarning(name, impl.Target()))
		}
	}
}

func (b *Bridge) storeConfig(conf Configuration) error {
	for key, val := range conf {
		storeKey := fmt.Sprintf("git-bug.bridge.%s.%s", b.Name, key)

		err := b.repo.LocalConfig().StoreString(storeKey, val)
		if err != nil {
			return errors.Wrap(err, "error while storing bridge configuration")
		}
	}

	return nil
}

func (b *Bridge) ensureConfig() error {
	if b.conf == nil {
		conf, err := loadConfig(b.repo, b.Name)
		if err != nil {
			return err
		}
		b.conf = conf
	}

	return nil
}

func loadConfig(repo repository.RepoConfig, name string) (Configuration, error) {
	keyPrefix := fmt.Sprintf("git-bug.bridge.%s.", name)

	pairs, err := repo.LocalConfig().ReadAll(keyPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "error while reading bridge configuration")
	}

	result := make(Configuration, len(pairs))
	for key, value := range pairs {
		key := strings.TrimPrefix(key, keyPrefix)
		result[key] = value
	}

	return result, nil
}

func (b *Bridge) getImporter() Importer {
	if b.importer == nil {
		b.importer = b.impl.NewImporter()
	}

	return b.importer
}

func (b *Bridge) getExporter() Exporter {
	if b.exporter == nil {
		b.exporter = b.impl.NewExporter()
	}

	return b.exporter
}

func (b *Bridge) ensureImportInit(ctx context.Context) error {
	if b.initImportDone {
		return nil
	}

	importer := b.getImporter()
	if importer != nil {
		err := importer.Init(ctx, b.repo, b.conf)
		if err != nil {
			return err
		}
	}

	b.initImportDone = true
	return nil
}

func (b *Bridge) ensureExportInit(ctx context.Context) error {
	if b.initExportDone {
		return nil
	}

	exporter := b.getExporter()
	if exporter != nil {
		err := exporter.Init(ctx, b.repo, b.conf)
		if err != nil {
			return err
		}
	}

	b.initExportDone = true
	return nil
}

func (b *Bridge) ImportAllSince(ctx context.Context, since time.Time) (<-chan ImportResult, error) {
	// 5 seconds before the actual start just to be sure.
	importStartTime := time.Now().Add(-5 * time.Second)

	importer := b.getImporter()
	if importer == nil {
		return nil, ErrImportNotSupported
	}

	err := b.ensureConfig()
	if err != nil {
		return nil, err
	}

	err = b.ensureImportInit(ctx)
	if err != nil {
		return nil, err
	}

	events, err := importer.ImportAll(ctx, b.repo, since)
	if err != nil {
		return nil, err
	}

	out := make(chan ImportResult)
	go func() {
		defer close(out)
		noError := true

		// relay all events while checking that everything went well
		for event := range events {
			if event.Event == ImportEventError {
				noError = false
			}
			out <- event
		}

		// store the last import time ONLY if no error happened
		if noError {
			key := fmt.Sprintf("git-bug.bridge.%s.lastImportTime", b.Name)
			err = b.repo.LocalConfig().StoreTimestamp(key, importStartTime)
		}
	}()

	return out, nil
}

func (b *Bridge) ImportAll(ctx context.Context) (<-chan ImportResult, error) {
	// If possible, restart from the last import time
	lastImport, err := b.repo.LocalConfig().ReadTimestamp(fmt.Sprintf("git-bug.bridge.%s.lastImportTime", b.Name))
	if err == nil {
		return b.ImportAllSince(ctx, lastImport)
	}

	return b.ImportAllSince(ctx, time.Time{})
}

func (b *Bridge) ExportAll(ctx context.Context, since time.Time) (<-chan ExportResult, error) {
	exporter := b.getExporter()
	if exporter == nil {
		return nil, ErrExportNotSupported
	}

	err := b.ensureConfig()
	if err != nil {
		return nil, err
	}

	err = b.ensureExportInit(ctx)
	if err != nil {
		return nil, err
	}

	return exporter.ExportAll(ctx, b.repo, since)
}
