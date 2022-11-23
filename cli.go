package main

import (
	"fmt"
	"strings"

	"github.com/c-bata/go-prompt"
)

type CLI struct {
}

type InGameStation struct {
	ID   string
	Name string
}

const (
	StationTypeStatic = "static"
	StationTypeStream = "stream"
)

const (
	CreateCmd  = "create"
	PlayCmd    = "play"
	DeleteCmd  = "delete"
	BindCmd    = "bind"
	BindAllCmd = "bindall"
	UnbindCmd  = "unbind"
	BindsCmd   = "binds"
)

var inGameStations = []InGameStation{
	{ID: "saeOLZXrNKpBEPGRBQ", Name: "Icon Radio"},
	{ID: "hgsuJcchvKuaEzzijr", Name: "Rock & Royale"},
	{ID: "VlYSRdFWOKyyhNNNgr", Name: "Radio Underground"},
	{ID: "DGeVaWdcXtfpbAaP", Name: "Party Royale"},
	{ID: "GEviYjIhzVVzJufW", Name: "Radio Yonder"},
	{ID: "BXrDueZkosvNvxtx", Name: "Beat Box"},
	{ID: "PcQCHxHkBsmjSneR", Name: "Power Play"},
}

func getInGameStationByName(name string) (InGameStation, bool) {
	for _, station := range inGameStations {
		if station.Name == name {
			return station, true
		}
	}

	return InGameStation{}, false
}

func (cli *CLI) completer(d prompt.Document) []prompt.Suggest {
	split := strings.Split(d.CurrentLine(), " ")

	var s []prompt.Suggest

	index := 0

	if len(split) == 1 {
		s = append(s, prompt.Suggest{Text: CreateCmd, Description: "Create a station"})
		s = append(s, prompt.Suggest{Text: PlayCmd, Description: "Plays a song on a station"})
		s = append(s, prompt.Suggest{Text: DeleteCmd, Description: "Deletes a station"})
		s = append(s, prompt.Suggest{Text: BindCmd, Description: "Bind a station to an in-game station"})
		s = append(s, prompt.Suggest{Text: BindAllCmd, Description: "Bind a station to all in-game stations"})
		s = append(s, prompt.Suggest{Text: BindsCmd, Description: "Lists all bound stations"})
		s = append(s, prompt.Suggest{Text: UnbindCmd, Description: "Unbind an in-game station"})
	}

	if len(split) == 2 && split[0] == CreateCmd {
		index = 1

		s = append(s, prompt.Suggest{Text: split[1], Description: "The station ID"})
	}

	if len(split) == 3 && split[0] == CreateCmd {
		index = 2

		s = append(s, prompt.Suggest{Text: StationTypeStatic})
		s = append(s, prompt.Suggest{Text: StationTypeStream})
	}

	if len(split) == 2 && (split[0] == PlayCmd || split[0] == DeleteCmd || split[0] == BindCmd || split[0] == BindAllCmd) {
		index = 1

		for _, station := range client.Users[client.APIClient.ID].Stations {
			s = append(s, prompt.Suggest{Text: station.ID})
		}
	}

	if len(split) >= 3 && split[0] == BindCmd {
		index = 2

		for _, station := range inGameStations {
			s = append(s, prompt.Suggest{Text: station.Name})
		}
	}

	if len(split) >= 2 && split[0] == UnbindCmd {
		index = 1

		for _, station := range inGameStations {
			if _, ok := client.Users[client.APIClient.ID].Bindings[station.ID]; ok {
				s = append(s, prompt.Suggest{Text: station.Name})
			}
		}
	}

	return prompt.FilterHasPrefix(s, strings.Join(split[index:], " "), true)
}

func (cli *CLI) createCmd(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: create <id> <type>")
		return
	}

	if _, ok := client.Users[client.APIClient.ID].Stations[args[0]]; ok {
		fmt.Println("Station already exists")
		return
	}

	station := APIStation{
		ID:   args[0],
		Type: args[1],
	}

	switch args[1] {
	case StationTypeStatic:
		if len(args) < 3 {
			fmt.Println("Usage: create <id> static <folder>")
			return
		}

		station.Source = strings.Join(args[2:], " ")
	case StationTypeStream:
		break
	default:
		fmt.Println("Unknown station type")
		return
	}

	err := client.APIClient.CreateStation(station)
	if err != nil {
		fmt.Println(err)
		return
	}

	client.Users[client.APIClient.ID].Stations[station.ID] = station

	fmt.Printf("Successfully created station %s\n", station.ID)
}

func (cli *CLI) playCmd(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: play <station> <song>")
		return
	}

	station, ok := client.Users[client.APIClient.ID].Stations[args[0]]
	if !ok {
		fmt.Println("Station not found")
		return
	}

	source := strings.Join(args[1:], " ")

	if station.Type == StationTypeStatic {
		err := client.APIClient.CreateStation(APIStation{
			ID:     station.ID,
			Type:   station.Type,
			Source: source,
		})

		if err != nil {
			fmt.Println(err)

			return
		}
	}

	if station.Type == StationTypeStream {
		err := client.APIClient.AddToQueue(station, source)

		if err != nil {
			fmt.Println(err)

			return
		}
	}

	fmt.Printf("%s is now playing %s\n", station.ID, source)
}

func (cli *CLI) deleteCmd(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: delete <station>")
		return
	}

	station, ok := client.Users[client.APIClient.ID].Stations[args[0]]
	if !ok {
		fmt.Println("Station not found")
		return
	}

	err := client.APIClient.DeleteStation(station)

	if err != nil {
		fmt.Println(err)

		return
	}

	delete(client.Users[client.APIClient.ID].Stations, station.ID)

	for i, binding := range client.Users[client.APIClient.ID].Bindings {
		if binding.StationUser == client.APIClient.ID && binding.StationID == station.ID {
			delete(client.Users[client.APIClient.ID].Bindings, i)
		}
	}
}

func (cli *CLI) bindCmd(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bind <station> <in-game station>")
		return
	}

	station, ok := client.Users[client.APIClient.ID].Stations[args[0]]
	if !ok {
		fmt.Println("Invalid station")
		return
	}

	inGameStation, ok := getInGameStationByName(strings.Join(args[1:], " "))

	if !ok {
		fmt.Println("In-game station not found")
		return
	}

	binding := APIBinding{
		ID:          inGameStation.ID,
		StationUser: client.APIClient.ID,
		StationID:   station.ID,
	}

	err := client.APIClient.CreateBinding(binding)
	if err != nil {
		fmt.Println(err)
		return
	}

	client.Users[client.APIClient.ID].Bindings[binding.ID] = binding

	fmt.Printf("Bound station %s to %s\n", station.ID, inGameStation.Name)
}

func (cli *CLI) bindAllCmd(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bindall <station>")
		return
	}

	station, ok := client.Users[client.APIClient.ID].Stations[args[0]]
	if !ok {
		fmt.Println("Invalid station")
		return
	}

	for _, inGameStation := range inGameStations {
		binding := APIBinding{
			ID:          inGameStation.ID,
			StationUser: client.APIClient.ID,
			StationID:   station.ID,
		}

		err := client.APIClient.CreateBinding(binding)
		if err != nil {
			fmt.Println(err)
			return
		}

		client.Users[client.APIClient.ID].Bindings[binding.ID] = binding

		fmt.Printf("Bound station %s to %s\n", station.ID, inGameStation.Name)
	}
}

func (cli *CLI) bindsCmd(_ []string) {
	for _, station := range inGameStations {
		if _, ok := client.Users[client.APIClient.ID].Bindings[station.ID]; ok {
			fmt.Printf("%s -> %s\n", station.Name, client.Users[client.APIClient.ID].Bindings[station.ID].StationID)
		} else {
			fmt.Printf("%s -> %s\n", station.Name, "Default")
		}
	}
}

func (cli *CLI) unbindCmd(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: unbind <in-game station>")
		return
	}

	name := strings.Join(args, " ")

	inGameStation, ok := getInGameStationByName(name)

	if ok {
		err := client.APIClient.DeleteBinding(APIBinding{ID: inGameStation.ID})
		if err != nil {
			fmt.Println(err)
			return
		}

		delete(client.Users[client.APIClient.ID].Bindings, inGameStation.ID)

		fmt.Printf("Unbound station %s\n", inGameStation.Name)
	} else {
		fmt.Println("In-game station not found")
	}
}

func (cli *CLI) execute(t string) {
	split := strings.Split(t, " ")

	switch split[0] {
	case CreateCmd:
		cli.createCmd(split[1:])
	case PlayCmd:
		cli.playCmd(split[1:])
	case DeleteCmd:
		cli.deleteCmd(split[1:])
	case BindCmd:
		cli.bindCmd(split[1:])
	case BindAllCmd:
		cli.bindAllCmd(split[1:])
	case UnbindCmd:
		cli.unbindCmd(split[1:])
	case BindsCmd:
		cli.bindsCmd(split[1:])
	default:
		fmt.Println("Unknown command")
	}
}

func setupCLI() {
	cli := &CLI{}

	p := prompt.New(cli.execute, cli.completer, prompt.OptionPrefix("> "))

	for {
		p.Run()
	}
}
