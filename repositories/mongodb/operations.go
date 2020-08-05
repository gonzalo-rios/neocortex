package mongodb

import (
	"context"
	"time"

	"fmt"

	"github.com/jinzhu/now"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/minskylab/neocortex"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
)

func (repo *Repository) SaveDialog(dialog *neocortex.Dialog) error {
	if dialog.ID == "" {
		dialog.ID = xid.New().String()
	}

	dialog.LastActivity = time.Now()

	_, err := repo.dialogs.InsertOne(context.Background(), dialog)
	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) GetDialogByID(id string) (*neocortex.Dialog, error) {
	dialog := new(neocortex.Dialog)
	if err := repo.dialogs.FindOne(context.Background(), bson.M{"id": id}).Decode(dialog); err != nil {
		return nil, err
	}

	return dialog, nil
}

func (repo *Repository) AllDialogs(frame neocortex.TimeFrame) ([]*neocortex.Dialog, error) {
	from := frame.From
	to := frame.To

	switch frame.Preset {
	case neocortex.DayPreset:
		from = now.BeginningOfDay()
		to = now.EndOfDay()
	case neocortex.WeekPreset:
		from = now.BeginningOfWeek()
		to = now.EndOfWeek()
	case neocortex.MonthPreset:
		from = now.BeginningOfMonth()
		to = now.EndOfMonth()
	case neocortex.YearPreset:
		from = now.BeginningOfYear()
		to = now.EndOfYear()
		// default:
		// 	return nil, errors.New("invalid preset, please choose between: day, week, month or year")
	}

	// * Mongo pleaseeee
	from = from.Truncate(time.Second)
	to = to.Truncate(time.Second)

	filter := bson.M{
		"start_at": bson.M{
			"$gte": primitive.DateTime(from.UnixNano() / 1000000),
			"$lte": primitive.DateTime(to.UnixNano() / 1000000),
		},
	}

	size := int64(frame.PageSize)
	if size <= 0 {
		size = 20
	}

	skips := int64(size * int64(frame.PageNum-1))
	if skips <= 0 {
		skips = 0
	}

	opts := options.Find().SetLimit(size).SetSkip(skips)

	cursor, err := repo.dialogs.Find(
		context.Background(),
		filter,
		opts,
	)

	if err != nil {
		return nil, err
	}

	dialogs := make([]*neocortex.Dialog, 0)
	for cursor.Next(context.Background()) {
		dialog := new(neocortex.Dialog)
		if err := cursor.Decode(dialog); err != nil {
			m := bson.M{}
			if err := cursor.Decode(&m); err != nil {
				continue
			}
			dialog.ID, _ = m["id"].(string)

			start, _ := m["start_at"].(primitive.DateTime)
			end, _ := m["end_at"].(primitive.DateTime)

			// * Mongo pleaseeee
			s := time.Unix(int64(start)/1000, int64(start)%1000*1000000)
			e := time.Unix(int64(end)/1000, int64(end)%1000*1000000)

			dialog.StartAt = s
			dialog.EndAt = e
			dialog.Ins = []*neocortex.InputRecord{}
			dialog.Outs = []*neocortex.OutputRecord{}
			dialog.Contexts = []*neocortex.ContextRecord{}
		}

		if dialog.Ins != nil && dialog.Outs != nil {
			if len(dialog.Ins) != 0 && len(dialog.Outs) != 0 {
				dialogs = append(dialogs, dialog)
			}
		}

	}

	return dialogs, nil
}

func (repo *Repository) DeleteDialog(id string) (*neocortex.Dialog, error) {
	dialog, err := repo.GetDialogByID(id)
	if err != nil {
		return nil, err
	}
	_, err = repo.dialogs.DeleteOne(context.Background(), bson.M{"id": id})
	if err != nil {
		return nil, err
	}

	return dialog, nil
}

func checkDialogInView(dialog *neocortex.Dialog, view *neocortex.View) bool {
	for _, c := range view.Classes {
		switch c.Type {
		case neocortex.EntityClass:
			if dialog.HasEntity(c.Value) {
				return true
			}
		case neocortex.IntentClass:
			if dialog.HasIntent(c.Value) {
				return true
			}
		case neocortex.DialogNodeClass:
			if dialog.HasDialogNode(c.Value) {
				return true
			}
		default:
			return false
		}
	}

	return false
}

func (repo *Repository) DialogsByView(viewID string, frame neocortex.TimeFrame) ([]*neocortex.Dialog, error) {
	view, err := repo.GetViewByID(viewID)
	if err != nil {
		return nil, err
	}

	dialogs, err := repo.AllDialogs(frame)
	if err != nil {
		return nil, err
	}

	filteredDialogs := make([]*neocortex.Dialog, 0)

	for _, dialog := range dialogs {
		if checkDialogInView(dialog, view) {
			filteredDialogs = append(filteredDialogs, dialog)
		}
	}

	return filteredDialogs, nil
}

func (repo *Repository) RegisterIntent(intent string) error {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "intents"}).Decode(coll)
	if err != nil {
		return err
	}

	for _, v := range coll.Values {
		if v == intent {
			return nil
		}
	}

	coll.Values = append(coll.Values, intent)

	_, err = repo.collections.UpdateOne(context.Background(), bson.M{"box": "intents"}, bson.M{"$set": bson.M{"values": coll.Values}})
	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) RegisterEntity(entity string) error {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "entities"}).Decode(coll)
	if err != nil {
		return err
	}

	for _, v := range coll.Values {
		if v == entity {
			return nil
		}
	}

	coll.Values = append(coll.Values, entity)

	_, err = repo.collections.UpdateOne(context.Background(), bson.M{"box": "entities"}, bson.M{"$set": bson.M{"values": coll.Values}})
	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) RegisterDialogNode(name string) error {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "nodes"}).Decode(coll)
	if err != nil {
		return err
	}

	for _, v := range coll.Values {
		if v == name {
			return nil
		}
	}

	coll.Values = append(coll.Values, name)

	_, err = repo.collections.UpdateOne(context.Background(), bson.M{"box": "nodes"}, bson.M{"$set": bson.M{"values": coll.Values}})
	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) RegisterContextVar(value string) error {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "context_vars"}).Decode(coll)
	if err != nil {
		return err
	}

	for _, v := range coll.Values {
		if v == value {
			return nil
		}
	}

	coll.Values = append(coll.Values, value)

	_, err = repo.collections.UpdateOne(context.Background(), bson.M{"box": "context_vars"}, bson.M{"$set": bson.M{"values": coll.Values}})
	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) Intents() []string {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "intents"}).Decode(coll)
	if err != nil {
		return nil
	}

	return coll.Values
}

func (repo *Repository) Entities() []string {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "entities"}).Decode(coll)
	if err != nil {
		return nil
	}

	return coll.Values
}

func (repo *Repository) DialogNodes() []string {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "nodes"}).Decode(coll)
	if err != nil {
		return nil
	}

	return coll.Values
}

func (repo *Repository) ContextVars() []string {
	coll := new(collection)
	err := repo.collections.FindOne(context.Background(), bson.M{"box": "context_vars"}).Decode(coll)
	if err != nil {
		return nil
	}

	return coll.Values
}

func (repo *Repository) SaveView(view *neocortex.View) error {
	if view.ID == "" {
		view.ID = xid.New().String()
	}

	_, err := repo.views.InsertOne(context.Background(), view)
	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) GetViewByID(id string) (*neocortex.View, error) {
	view := new(neocortex.View)
	if err := repo.views.FindOne(context.Background(), bson.M{"id": id}).Decode(view); err != nil {
		return nil, err
	}

	return view, nil
}

func (repo *Repository) FindViewByName(name string) ([]*neocortex.View, error) {

	c, err := repo.views.Find(context.Background(), bson.M{"name": name})
	if err != nil {
		return nil, err
	}

	views := make([]*neocortex.View, 0)
	if err != nil {
		return nil, err
	}

	for c.Next(context.Background()) {
		view := new(neocortex.View)
		if err := c.Decode(view); err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, nil
}

func (repo *Repository) AllViews() ([]*neocortex.View, error) {
	c, err := repo.views.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}

	views := make([]*neocortex.View, 0)
	if err != nil {
		return nil, err
	}

	for c.Next(context.Background()) {
		view := new(neocortex.View)
		if err := c.Decode(view); err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, nil
}

//! MAKE GOOD UPDATE IN UPDATE VIEW
func (repo *Repository) UpdateView(view *neocortex.View) error {
	return nil
	// _, err := repo.views.UpdateOne(context.Background(), bson.M{"id": view.ID}, bson.M{
	// 	"name":     view.Name,
	// 	"styles":   view.Styles,
	// 	"classes":  view.Classes,
	// 	"children": view.Children,
	// })

	// return err
}

func (repo *Repository) DeleteView(id string) (*neocortex.View, error) {
	view, err := repo.GetViewByID(id)
	if err != nil {
		return nil, err
	}

	del, err := repo.views.DeleteOne(context.Background(), bson.M{"id": view.ID})
	if err != nil {
		return nil, err
	}
	fmt.Println("del.DeletedCount =", del.DeletedCount)

	return view, nil

}

func (repo *Repository) SetActionVar(name string, value string) error {
	act := new(action)
	err := repo.actions.FindOne(context.Background(), bson.M{"name": "envs"}).Decode(act)
	if err != nil {
		return err
	}

	act.Vars[name] = value

	_, err = repo.actions.UpdateOne(context.Background(), bson.M{"name": "envs"}, bson.M{"$set": bson.M{"vars": act.Vars}})
	return err
}

func (repo *Repository) GetActionVar(name string) (string, error) {
	act := new(action)
	err := repo.actions.FindOne(context.Background(), bson.M{"name": "envs"}).Decode(act)
	if err != nil {
		return "", err
	}

	return act.Vars[name], nil
}

func (repo *Repository) Summary(frame neocortex.TimeFrame) (*neocortex.Summary, error) {

	summary := neocortex.Summary{}
	summary.UsersByTimezone = map[string]neocortex.UsersSummary{}

	from := frame.From
	to := frame.To

	switch frame.Preset {
	case neocortex.DayPreset:
		from = now.BeginningOfDay()
		to = now.EndOfDay()
	case neocortex.WeekPreset:
		from = now.BeginningOfWeek()
		to = now.EndOfWeek()
	case neocortex.MonthPreset:
		from = now.BeginningOfMonth()
		to = now.EndOfMonth()
	case neocortex.YearPreset:
		from = now.BeginningOfYear()
		to = now.EndOfYear()
		// default:
		// 	return nil, errors.New("invalid preset, please choose between: day, week, month or year")
	}

	// * Mongo pleaseeee

	filter := bson.M{
		"start_at": bson.M{
			"$gte": primitive.DateTime(from.UnixNano() / 1000000),
			"$lte": primitive.DateTime(to.UnixNano() / 1000000),
		},
	}

	fmt.Println(" ======== SUMMARY ======== ")
	fmt.Printf("From: %s\tTo: %s\n", from, to)

	opts := options.Find().SetLimit(1e8) // 100000000

	fmt.Printf("opts: %v\n", opts.Limit)

	cursor, err := repo.dialogs.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}

	performanceAccum := 0.0
	totalCorrectDialogs := int64(0)

	usersByTimezone := map[string]map[string]int{}

	for cursor.Next(context.Background()) {
		dialog := new(neocortex.Dialog)
		err = cursor.Decode(dialog)
		if err != nil {
			continue
		}

		if len(dialog.Contexts) > 0 {
			c := dialog.Contexts[0]
			if usersByTimezone[c.Context.Person.Timezone] == nil {
				usersByTimezone[c.Context.Person.Timezone] = map[string]int{}
			}
			usersByTimezone[c.Context.Person.Timezone][c.Context.Person.Name]++
		}

		if len(dialog.Ins) > 0 {
			performanceAccum += dialog.Performance
			totalCorrectDialogs++
		}
	}

	if totalCorrectDialogs == 0 {
		return &neocortex.Summary{
			TotalDialogs:    0,
			TotalUsers:      0,
			RecurrentUsers:  0,
			PerformanceMean: 0.0,
			// UsersByTimezone: map[string,
		}, nil
	}

	totalUsers := int64(0)
	totalRecurrent := int64(0)

	for timezone, totalUsersByTimezone := range usersByTimezone {
		recurrent := 0
		for _, r := range totalUsersByTimezone {
			if r > 1 {
				recurrent++
			}
		}

		summary.UsersByTimezone[timezone] = neocortex.UsersSummary{
			Recurrents: int64(recurrent),
			News:       int64(len(totalUsersByTimezone) - recurrent),
		}

		totalUsers += int64(len(totalUsersByTimezone))
		totalRecurrent += int64(recurrent)
	}

	summary.TotalUsers = totalUsers
	summary.RecurrentUsers = totalRecurrent
	summary.TotalDialogs = totalCorrectDialogs
	summary.PerformanceMean = performanceAccum / float64(totalCorrectDialogs)

	return &summary, nil
}
