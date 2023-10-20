package cache

import (
	"encoding/gob"
	"time"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// Package initialisation used to register the type for (de)serialization
func init() {
	gob.Register(BoardExcerpt{})
}

var _ Excerpt = &BoardExcerpt{}

// BoardExcerpt hold a subset of the board values to be able to sort and filter boards
// efficiently without having to read and compile each raw boards.
type BoardExcerpt struct {
	id entity.Id

	CreateLamportTime lamport.Time
	EditLamportTime   lamport.Time
	CreateUnixTime    int64
	EditUnixTime      int64

	Title        string
	Description  string
	ItemCount    int
	Participants []entity.Id

	CreateMetadata map[string]string
}

func NewBoardExcerpt(b *BoardCache) *BoardExcerpt {
	snap := b.Snapshot()

	participantsIds := make([]entity.Id, 0, len(snap.Participants))
	for _, participant := range snap.Participants {
		participantsIds = append(participantsIds, participant.Id())
	}

	return &BoardExcerpt{
		id:                b.Id(),
		CreateLamportTime: b.CreateLamportTime(),
		EditLamportTime:   b.EditLamportTime(),
		CreateUnixTime:    b.FirstOp().Time().Unix(),
		EditUnixTime:      snap.EditTime().Unix(),
		Title:             snap.Title,
		Description:       snap.Description,
		ItemCount:         snap.ItemCount(),
		Participants:      participantsIds,
		CreateMetadata:    b.FirstOp().AllMetadata(),
	}
}

func (b *BoardExcerpt) Id() entity.Id {
	return b.id
}

func (b *BoardExcerpt) setId(id entity.Id) {
	b.id = id
}

func (b *BoardExcerpt) CreateTime() time.Time {
	return time.Unix(b.CreateUnixTime, 0)
}

func (b *BoardExcerpt) EditTime() time.Time {
	return time.Unix(b.EditUnixTime, 0)
}
