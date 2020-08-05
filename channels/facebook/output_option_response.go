package facebook

import (
	"github.com/minskylab/neocortex"
	"github.com/minskylab/neocortex/channels/facebook/messenger"
)

func sendOneOptionResponse(userID int64, msn *messenger.Messenger, options neocortex.OptionsResponse) error {
	gm := msn.NewGenericMessage(userID)
	buttons := make([]messenger.Button, 0)
	for _, o := range options.Options {
		if o.IsPostBack {
			buttons = append(buttons, msn.NewPostbackButton(o.Text, o.Action))
		} else {
			buttons = append(buttons, msn.NewWebURLButton(o.Text, o.Action))
		}
	}
	gm.AddNewElement(options.Title, options.Description, options.ItemURL, options.Image, buttons)
	_, err := msn.SendMessage(gm)
	return err
}

func sendManyOptionsResponse(userID int64, msn *messenger.Messenger, optionsArray []neocortex.OptionsResponse) error {
	gm := msn.NewGenericMessage(userID)
	for _, options := range optionsArray {
		buttons := make([]messenger.Button, 0)
		for _, o := range options.Options {
			if o.IsPostBack {
				buttons = append(buttons, msn.NewPostbackButton(o.Text, o.Action))
			} else {
				buttons = append(buttons, msn.NewWebURLButton(o.Text, o.Action))
			}
		}
		gm.AddNewElement(options.Title, options.Description, options.ItemURL, options.Image, buttons)
	}
	_, err := msn.SendMessage(gm)
	return err
}
