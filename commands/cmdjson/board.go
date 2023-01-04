package cmdjson

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entities/board"
)

type BoardSnapshot struct {
	Id         string `json:"id"`
	HumanId    string `json:"human_id"`
	CreateTime Time   `json:"create_time"`
	EditTime   Time   `json:"edit_time"`

	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Participants []Identity    `json:"participants"`
	Columns      []BoardColumn `json:"columns"`
}

func NewBoardSnapshot(snapshot *board.Snapshot) BoardSnapshot {
	jsonBoard := BoardSnapshot{
		Id:          snapshot.Id().String(),
		HumanId:     snapshot.Id().Human(),
		CreateTime:  NewTime(snapshot.CreateTime, 0),
		EditTime:    NewTime(snapshot.EditTime(), 0),
		Title:       snapshot.Title,
		Description: snapshot.Description,
	}

	jsonBoard.Participants = make([]Identity, len(snapshot.Participants))
	for i, element := range snapshot.Participants {
		jsonBoard.Participants[i] = NewIdentity(element)
	}

	jsonBoard.Columns = make([]BoardColumn, len(snapshot.Columns))
	for i, column := range snapshot.Columns {
		jsonBoard.Columns[i] = NewBoardColumn(column)
	}

	return jsonBoard
}

type BoardColumn struct {
	Id      string `json:"id"`
	HumanId string `json:"human_id"`
	Name    string `json:"name"`
	Items   []any  `json:"items"`
}

func NewBoardColumn(column *board.Column) BoardColumn {
	jsonColumn := BoardColumn{
		Id:      column.CombinedId.String(),
		HumanId: column.CombinedId.Human(),
		Name:    column.Name,
	}
	jsonColumn.Items = make([]any, len(column.Items))
	for j, item := range column.Items {
		switch item := item.(type) {
		case *board.Draft:
			jsonColumn.Items[j] = NewBoardDraftItem(item)
		case *board.BugItem:
			jsonColumn.Items[j] = NewBugSnapshot(item.Bug.Compile())
		default:
			panic("unknown item type")
		}
	}
	return jsonColumn
}

type BoardDraftItem struct {
	Id      string   `json:"id"`
	HumanId string   `json:"human_id"`
	Author  Identity `json:"author"`
	Title   string   `json:"title"`
	Message string   `json:"message"`
}

func NewBoardDraftItem(item *board.Draft) BoardDraftItem {
	return BoardDraftItem{
		Id:      item.CombinedId().String(),
		HumanId: item.CombinedId().Human(),
		Author:  NewIdentity(item.Author),
		Title:   item.Title,
		Message: item.Message,
	}
}

type BoardExcerpt struct {
	Id         string `json:"id"`
	HumanId    string `json:"human_id"`
	CreateTime Time   `json:"create_time"`
	EditTime   Time   `json:"edit_time"`

	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Participants []Identity `json:"participants"`

	Items    int               `json:"items"`
	Metadata map[string]string `json:"metadata"`
}

func NewBoardExcerpt(backend *cache.RepoCache, b *cache.BoardExcerpt) (BoardExcerpt, error) {
	jsonBoard := BoardExcerpt{
		Id:          b.Id().String(),
		HumanId:     b.Id().Human(),
		CreateTime:  NewTime(b.CreateTime(), b.CreateLamportTime),
		EditTime:    NewTime(b.EditTime(), b.EditLamportTime),
		Title:       b.Title,
		Description: b.Description,
		Items:       b.ItemCount,
		Metadata:    b.CreateMetadata,
	}

	jsonBoard.Participants = make([]Identity, len(b.Participants))
	for i, element := range b.Participants {
		participant, err := backend.Identities().ResolveExcerpt(element)
		if err != nil {
			return BoardExcerpt{}, err
		}
		jsonBoard.Participants[i] = NewIdentityFromExcerpt(participant)
	}
	return jsonBoard, nil
}
