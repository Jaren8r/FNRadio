package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"jaren.wtf/fnradio/client/pkg/logreader"
)

var onPartyCreate = regexp.MustCompile(`LogOnlineParty: MCP: OnCreatePartyComplete: User=\[([0-9a-f]{32})] Party=\[(V2:[0-9a-f]{32})]`)
var onPartyJoin = regexp.MustCompile(`LogOnlineParty: MCP: JoinParty: User=\[([0-9a-f]{32})] .+PartyId\((V2:[0-9a-f]{32})\)`)
var onPartyNewLeader = regexp.MustCompile(`LogOnlineParty: MCP: OnPartyNewLeader: User=\[([0-9a-f]{32})] Party=\[(V2:[0-9a-f]{32})] NewLeader=\[([0-9a-f]{32}|[0-9a-f]{5}\.\.\.[0-9a-f]{5})]`)
var onPartyMemberPromoted = regexp.MustCompile(`LogParty: Verbose: Member \[MCP:([0-9a-f]{32}|[0-9a-f]{5}\.\.\.[0-9a-f]{5}), Party \((V2:[0-9a-f]{32})\)] promoted to party leader`)
var onMatchmakingSession = regexp.MustCompile(`LogMatchmakingServiceClient: Verbose: HandleWebSocketMessage - Received message: "{"payload":{"matchId":"([0-9a-f]{32})","sessionId":"([0-9a-f]{32})","joinDelaySec":\d+},"name":"Play"}"`)

type Party struct {
	ID      string `json:"id"`
	Match   string `json:"match"`
	Session string `json:"session"`
	Leader  bool   `json:"leader"`
}

func (party *Party) Equals(party2 Party) bool {
	return party.ID == party2.ID && party.Leader == party2.Leader && party.Match == party2.Match && party.Session == party2.Session
}

func (client *FNRadioClient) handlePartyChange(oldParty Party, newParty Party) {
	if oldParty.Match != newParty.Match {
		for i := 1; i < 5; i++ {
			leader, err := client.APIClient.SetParty(newParty)
			if err == nil {
				if leader != "" {
					_ = client.Logger.Output(2, "Successfully set FNRadio party with leader "+leader)
				} else {
					_ = client.Logger.Output(2, "Successfully disabled FNRadio party")
				}

				if leader != client.APIClient.ID {
					_ = client.Logger.Output(2, "Fetching party leader "+leader)

					client.Users[leader], err = client.APIClient.GetUser(leader)
					if err != nil {
						_ = client.Logger.Output(2, "Error fetching party leader: "+err.Error())
					} else {
						client.BoundUser = leader
					}
				} else if client.BoundUser != client.APIClient.ID {
					delete(client.Users, client.BoundUser)
					client.BoundUser = client.APIClient.ID
				}

				break
			}

			_ = client.Logger.Output(2, "Failed to set party: "+err.Error())

			time.Sleep(time.Second)
		}
	}
}

func (client *FNRadioClient) handleGameLogLines(lines []string) { // nolint:funlen
	old := client.Party

	for _, line := range lines {
		if strings.Contains(line, "LogOnlineGame: FortPC::ReturnToMainMenu()") {
			client.Party.Match = ""
			client.Party.Session = ""

			_ = client.Logger.Output(2, "Reset match info as we have returned to main menu")

			continue
		}

		onCreate := onPartyCreate.FindStringSubmatch(line)
		if len(onCreate) != 0 {
			client.Party.ID = onCreate[2]
			client.Party.Leader = true

			_ = client.Logger.Output(2, "Created party "+client.Party.ID)

			continue
		}

		onJoin := onPartyJoin.FindStringSubmatch(line)
		if len(onJoin) != 0 {
			client.Party.ID = onJoin[2]
			client.Party.Leader = false

			_ = client.Logger.Output(2, "Joined party "+client.Party.ID)

			continue
		}

		onNewLeader := onPartyNewLeader.FindStringSubmatch(line)
		if len(onNewLeader) != 0 {
			client.Party.ID = onNewLeader[2]
			client.Party.Leader = !strings.Contains(onNewLeader[3], "...")

			_ = client.Logger.Output(2, "Updated party leader status to "+strconv.FormatBool(client.Party.Leader))

			continue
		}

		onMemberPromoted := onPartyMemberPromoted.FindStringSubmatch(line)
		if len(onMemberPromoted) != 0 {
			client.Party.ID = onMemberPromoted[2]
			client.Party.Leader = !strings.Contains(onMemberPromoted[1], "...")

			_ = client.Logger.Output(2, "Updated party leader status to "+strconv.FormatBool(client.Party.Leader))

			continue
		}

		onSession := onMatchmakingSession.FindStringSubmatch(line)
		if len(onSession) != 0 {
			client.Party.Match = onSession[1]
			client.Party.Session = onSession[2]

			_ = client.Logger.Output(2, "Joined match "+client.Party.Match+"/"+client.Party.Session)

			continue
		}
	}

	if !old.Equals(client.Party) {
		client.handlePartyChange(old, client.Party)
	}
}

func (client *FNRadioClient) readGameLog() {
	reader, err := logreader.New(os.Getenv("LOCALAPPDATA") + `\FortniteGame\Saved\Logs\FortniteGame.log`)
	if err != nil {
		panic(err)
	}

	for event := range reader.Events {
		if event.Error != nil {
			fmt.Println(event.Error)
			break
		}

		if event.Initial && strings.Contains(event.Lines[len(event.Lines)-1], "Log file closed, ") {
			continue // The game isn't open right now, so we really don't care what the logs say is happening
		}

		client.handleGameLogLines(event.Lines)
	}
}
