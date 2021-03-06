// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/store"
	"github.com/stretchr/testify/assert"
)

func TestPermanentDeleteChannel(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		cfg.ServiceSettings.EnableIncomingWebhooks = true
		cfg.ServiceSettings.EnableOutgoingWebhooks = true
	})

	channel, err := th.App.CreateChannel(&model.Channel{DisplayName: "deletion-test", Name: "deletion-test", Type: model.CHANNEL_OPEN, TeamId: th.BasicTeam.Id}, false)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer func() {
		th.App.PermanentDeleteChannel(channel)
	}()

	incoming, err := th.App.CreateIncomingWebhookForChannel(th.BasicUser.Id, channel, &model.IncomingWebhook{ChannelId: channel.Id})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer th.App.DeleteIncomingWebhook(incoming.Id)

	if incoming, err = th.App.GetIncomingWebhook(incoming.Id); incoming == nil || err != nil {
		t.Fatal("unable to get new incoming webhook")
	}

	outgoing, err := th.App.CreateOutgoingWebhook(&model.OutgoingWebhook{
		ChannelId:    channel.Id,
		TeamId:       channel.TeamId,
		CreatorId:    th.BasicUser.Id,
		CallbackURLs: []string{"http://foo"},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer th.App.DeleteOutgoingWebhook(outgoing.Id)

	if outgoing, err = th.App.GetOutgoingWebhook(outgoing.Id); outgoing == nil || err != nil {
		t.Fatal("unable to get new outgoing webhook")
	}

	if err := th.App.PermanentDeleteChannel(channel); err != nil {
		t.Fatal(err.Error())
	}

	if incoming, err = th.App.GetIncomingWebhook(incoming.Id); incoming != nil || err == nil {
		t.Error("incoming webhook wasn't deleted")
	}

	if outgoing, err = th.App.GetOutgoingWebhook(outgoing.Id); outgoing != nil || err == nil {
		t.Error("outgoing webhook wasn't deleted")
	}
}

func TestMoveChannel(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	sourceTeam := th.CreateTeam()
	targetTeam := th.CreateTeam()
	channel1 := th.CreateChannel(sourceTeam)
	defer func() {
		th.App.PermanentDeleteChannel(channel1)
		th.App.PermanentDeleteTeam(sourceTeam)
		th.App.PermanentDeleteTeam(targetTeam)
	}()

	if _, err := th.App.AddUserToTeam(sourceTeam.Id, th.BasicUser.Id, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := th.App.AddUserToTeam(sourceTeam.Id, th.BasicUser2.Id, ""); err != nil {
		t.Fatal(err)
	}

	if _, err := th.App.AddUserToTeam(targetTeam.Id, th.BasicUser.Id, ""); err != nil {
		t.Fatal(err)
	}

	if _, err := th.App.AddUserToChannel(th.BasicUser, channel1); err != nil {
		t.Fatal(err)
	}
	if _, err := th.App.AddUserToChannel(th.BasicUser2, channel1); err != nil {
		t.Fatal(err)
	}

	if err := th.App.MoveChannel(targetTeam, channel1, th.BasicUser); err == nil {
		t.Fatal("Should have failed due to mismatched members.")
	}

	if _, err := th.App.AddUserToTeam(targetTeam.Id, th.BasicUser2.Id, ""); err != nil {
		t.Fatal(err)
	}

	if err := th.App.MoveChannel(targetTeam, channel1, th.BasicUser); err != nil {
		t.Fatal(err)
	}
}

func TestJoinDefaultChannelsCreatesChannelMemberHistoryRecordTownSquare(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	// figure out the initial number of users in town square
	townSquareChannelId := store.Must(th.App.Srv.Store.Channel().GetByName(th.BasicTeam.Id, "town-square", true)).(*model.Channel).Id
	initialNumTownSquareUsers := len(store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, townSquareChannelId)).([]*model.ChannelMemberHistoryResult))

	// create a new user that joins the default channels
	user := th.CreateUser()
	th.App.JoinDefaultChannels(th.BasicTeam.Id, user, model.CHANNEL_USER_ROLE_ID, "")

	// there should be a ChannelMemberHistory record for the user
	histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, townSquareChannelId)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, initialNumTownSquareUsers+1)

	found := false
	for _, history := range histories {
		if user.Id == history.UserId && townSquareChannelId == history.ChannelId {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestJoinDefaultChannelsCreatesChannelMemberHistoryRecordOffTopic(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	// figure out the initial number of users in off-topic
	offTopicChannelId := store.Must(th.App.Srv.Store.Channel().GetByName(th.BasicTeam.Id, "off-topic", true)).(*model.Channel).Id
	initialNumTownSquareUsers := len(store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, offTopicChannelId)).([]*model.ChannelMemberHistoryResult))

	// create a new user that joins the default channels
	user := th.CreateUser()
	th.App.JoinDefaultChannels(th.BasicTeam.Id, user, model.CHANNEL_USER_ROLE_ID, "")

	// there should be a ChannelMemberHistory record for the user
	histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, offTopicChannelId)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, initialNumTownSquareUsers+1)

	found := false
	for _, history := range histories {
		if user.Id == history.UserId && offTopicChannelId == history.ChannelId {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestCreateChannelPublicCreatesChannelMemberHistoryRecord(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	// creates a public channel and adds basic user to it
	publicChannel := th.createChannel(th.BasicTeam, model.CHANNEL_OPEN)

	// there should be a ChannelMemberHistory record for the user
	histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, publicChannel.Id)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, 1)
	assert.Equal(t, th.BasicUser.Id, histories[0].UserId)
	assert.Equal(t, publicChannel.Id, histories[0].ChannelId)
}

func TestCreateChannelPrivateCreatesChannelMemberHistoryRecord(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	// creates a private channel and adds basic user to it
	privateChannel := th.createChannel(th.BasicTeam, model.CHANNEL_PRIVATE)

	// there should be a ChannelMemberHistory record for the user
	histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, privateChannel.Id)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, 1)
	assert.Equal(t, th.BasicUser.Id, histories[0].UserId)
	assert.Equal(t, privateChannel.Id, histories[0].ChannelId)
}

func TestUpdateChannelPrivacy(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	privateChannel := th.createChannel(th.BasicTeam, model.CHANNEL_PRIVATE)
	privateChannel.Type = model.CHANNEL_OPEN

	if publicChannel, err := th.App.UpdateChannelPrivacy(privateChannel, th.BasicUser); err != nil {
		t.Fatal("Failed to update channel privacy. Error: " + err.Error())
	} else {
		assert.Equal(t, publicChannel.Id, privateChannel.Id)
		assert.Equal(t, publicChannel.Type, model.CHANNEL_OPEN)
	}
}

func TestCreateGroupChannelCreatesChannelMemberHistoryRecord(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	user1 := th.CreateUser()
	user2 := th.CreateUser()

	groupUserIds := make([]string, 0)
	groupUserIds = append(groupUserIds, user1.Id)
	groupUserIds = append(groupUserIds, user2.Id)
	groupUserIds = append(groupUserIds, th.BasicUser.Id)

	if channel, err := th.App.CreateGroupChannel(groupUserIds, th.BasicUser.Id); err != nil {
		t.Fatal("Failed to create group channel. Error: " + err.Message)
	} else {
		// there should be a ChannelMemberHistory record for each user
		histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, channel.Id)).([]*model.ChannelMemberHistoryResult)
		assert.Len(t, histories, 3)

		channelMemberHistoryUserIds := make([]string, 0)
		for _, history := range histories {
			assert.Equal(t, channel.Id, history.ChannelId)
			channelMemberHistoryUserIds = append(channelMemberHistoryUserIds, history.UserId)
		}
		assert.Equal(t, groupUserIds, channelMemberHistoryUserIds)
	}
}

func TestCreateDirectChannelCreatesChannelMemberHistoryRecord(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	user1 := th.CreateUser()
	user2 := th.CreateUser()

	if channel, err := th.App.CreateDirectChannel(user1.Id, user2.Id); err != nil {
		t.Fatal("Failed to create direct channel. Error: " + err.Message)
	} else {
		// there should be a ChannelMemberHistory record for both users
		histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, channel.Id)).([]*model.ChannelMemberHistoryResult)
		assert.Len(t, histories, 2)

		historyId0 := histories[0].UserId
		historyId1 := histories[1].UserId
		switch historyId0 {
		case user1.Id:
			assert.Equal(t, user2.Id, historyId1)
		case user2.Id:
			assert.Equal(t, user1.Id, historyId1)
		default:
			t.Fatal("Unexpected user id " + historyId0 + " in ChannelMemberHistory table")
		}
	}
}

func TestGetDirectChannelCreatesChannelMemberHistoryRecord(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	user1 := th.CreateUser()
	user2 := th.CreateUser()

	// this function call implicitly creates a direct channel between the two users if one doesn't already exist
	if channel, err := th.App.GetDirectChannel(user1.Id, user2.Id); err != nil {
		t.Fatal("Failed to create direct channel. Error: " + err.Message)
	} else {
		// there should be a ChannelMemberHistory record for both users
		histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, channel.Id)).([]*model.ChannelMemberHistoryResult)
		assert.Len(t, histories, 2)

		historyId0 := histories[0].UserId
		historyId1 := histories[1].UserId
		switch historyId0 {
		case user1.Id:
			assert.Equal(t, user2.Id, historyId1)
		case user2.Id:
			assert.Equal(t, user1.Id, historyId1)
		default:
			t.Fatal("Unexpected user id " + historyId0 + " in ChannelMemberHistory table")
		}
	}
}

func TestAddUserToChannelCreatesChannelMemberHistoryRecord(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	// create a user and add it to a channel
	user := th.CreateUser()
	if _, err := th.App.AddTeamMember(th.BasicTeam.Id, user.Id); err != nil {
		t.Fatal("Failed to add user to team. Error: " + err.Message)
	}

	groupUserIds := make([]string, 0)
	groupUserIds = append(groupUserIds, th.BasicUser.Id)
	groupUserIds = append(groupUserIds, user.Id)

	channel := th.createChannel(th.BasicTeam, model.CHANNEL_OPEN)
	if _, err := th.App.AddUserToChannel(user, channel); err != nil {
		t.Fatal("Failed to add user to channel. Error: " + err.Message)
	}

	// there should be a ChannelMemberHistory record for the user
	histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, channel.Id)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, 2)
	channelMemberHistoryUserIds := make([]string, 0)
	for _, history := range histories {
		assert.Equal(t, channel.Id, history.ChannelId)
		channelMemberHistoryUserIds = append(channelMemberHistoryUserIds, history.UserId)
	}
	assert.Equal(t, groupUserIds, channelMemberHistoryUserIds)
}

func TestRemoveUserFromChannelUpdatesChannelMemberHistoryRecord(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	// a user creates a channel
	publicChannel := th.createChannel(th.BasicTeam, model.CHANNEL_OPEN)
	histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, publicChannel.Id)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, 1)
	assert.Equal(t, th.BasicUser.Id, histories[0].UserId)
	assert.Equal(t, publicChannel.Id, histories[0].ChannelId)
	assert.Nil(t, histories[0].LeaveTime)

	// the user leaves that channel
	if err := th.App.LeaveChannel(publicChannel.Id, th.BasicUser.Id); err != nil {
		t.Fatal("Failed to remove user from channel. Error: " + err.Message)
	}
	histories = store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, publicChannel.Id)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, 1)
	assert.Equal(t, th.BasicUser.Id, histories[0].UserId)
	assert.Equal(t, publicChannel.Id, histories[0].ChannelId)
	assert.NotNil(t, histories[0].LeaveTime)
}

func TestAddChannelMemberNoUserRequestor(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	// create a user and add it to a channel
	user := th.CreateUser()
	if _, err := th.App.AddTeamMember(th.BasicTeam.Id, user.Id); err != nil {
		t.Fatal("Failed to add user to team. Error: " + err.Message)
	}

	groupUserIds := make([]string, 0)
	groupUserIds = append(groupUserIds, th.BasicUser.Id)
	groupUserIds = append(groupUserIds, user.Id)

	channel := th.createChannel(th.BasicTeam, model.CHANNEL_OPEN)
	userRequestorId := ""
	postRootId := ""
	if _, err := th.App.AddChannelMember(user.Id, channel, userRequestorId, postRootId); err != nil {
		t.Fatal("Failed to add user to channel. Error: " + err.Message)
	}

	// there should be a ChannelMemberHistory record for the user
	histories := store.Must(th.App.Srv.Store.ChannelMemberHistory().GetUsersInChannelDuring(model.GetMillis()-100, model.GetMillis()+100, channel.Id)).([]*model.ChannelMemberHistoryResult)
	assert.Len(t, histories, 2)
	channelMemberHistoryUserIds := make([]string, 0)
	for _, history := range histories {
		assert.Equal(t, channel.Id, history.ChannelId)
		channelMemberHistoryUserIds = append(channelMemberHistoryUserIds, history.UserId)
	}
	assert.Equal(t, groupUserIds, channelMemberHistoryUserIds)

	postList := store.Must(th.App.Srv.Store.Post().GetPosts(channel.Id, 0, 1, false)).(*model.PostList)
	if assert.Len(t, postList.Order, 1) {
		post := postList.Posts[postList.Order[0]]

		assert.Equal(t, model.POST_JOIN_CHANNEL, post.Type)
		assert.Equal(t, user.Id, post.UserId)
		assert.Equal(t, user.Username, post.Props["username"])
	}
}
