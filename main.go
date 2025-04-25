package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/placki-w/pokedexcli/internal/pokecache"
)

var commands map[string]cliCommand
var cfg config
var pokeCache *pokecache.Cache
var pokedex map[string]pokemonDetails

func main() {

	//base url for poke location area
	cfg.nextUrl = "https://pokeapi.co/api/v2/location-area/"

	// Create a new cache that expires items after 5 minutes
	pokeCache = pokecache.NewCache(5 * time.Minute)

	// The pokedex, the thing we want
	pokedex = make(map[string]pokemonDetails)

	// Define the commands map
	commands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"map": {
			name:        "map",
			description: "Displays the next 20 locations in the Pokemon world",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Displays the previous 20 locations in the Pokemon world",
			callback:    commandMapb,
		},
		"explore": {
			name:        "explore",
			description: "Provides more details at a given location",
			callback:    commandExplore,
		},
		"catch": {
			name:        "catch",
			description: "Attempt to catch a pokemon based on chance and return details if so",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "Reveal high level details of your captured pokemon",
			callback:    commandInspect,
		},
		"pokedex": {
			name:        "pokedex",
			description: "list all your captured pokemon",
			callback:    commandPokedex,
		},
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		input := scanner.Text()

		// Split the input by spaces to separate command and arguments
		args := strings.Fields(input)
		if len(args) == 0 {
			fmt.Println("Please input a command.")
			continue
		}

		commandName := strings.ToLower(args[0])

		// Check if the command exists
		cmd, ok := commands[commandName]
		if !ok {
			fmt.Println("Unknown command")
			continue
		}

		// Pass any arguments after the command name
		var cmdArgs []string
		if len(args) > 1 {
			cmdArgs = args[1:]
		}

		// Call the command with its arguments
		err := cmd.callback(&cfg, pokeCache, cmdArgs...)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func cleanInput(text string) string {
	// Clean input by trimming spaces and converting to lowercase
	trimmedInput := strings.TrimSpace(strings.ToLower(text))
	return trimmedInput
}

func commandExit(cfg *config, cache *pokecache.Cache, area ...string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config, cache *pokecache.Cache, area ...string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage: ")
	fmt.Println("")
	for _, cmd := range commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func commandMap(cfg *config, cache *pokecache.Cache, area ...string) error {

	url := cfg.nextUrl
	if cfg.nextUrl != "" {
		url = cfg.nextUrl
	}

	var body []byte
	var err error

	cachedData, found := cache.Get(url)
	if found {
		fmt.Println("Using cached data for:", url) //optional logging
		body = cachedData
	} else {
		//not in cache make HTTP request
		fmt.Println("Fetching new data for:", url) //optional logging
		res, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		body, err = io.ReadAll(res.Body)
		defer res.Body.Close()
		if res.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
		}

		if err != nil {
			log.Fatal(err)
		}

		// Add to cache
		cache.Add(url, body)

	}
	var locations responseBody
	if err = json.Unmarshal(body, &locations); err != nil {
		return err
	}

	for _, location := range locations.Results {
		fmt.Println(location.Name)
	}

	cfg.nextUrl = locations.Next
	if locations.Previous != nil {
		cfg.previousUrl = locations.Previous.(string)
	} else {
		cfg.previousUrl = ""
	}

	return nil
}

func commandMapb(cfg *config, cache *pokecache.Cache, area ...string) error {
	if cfg.previousUrl == "" {
		fmt.Println("you're on the first page")
		return nil
	}

	var body []byte
	var err error

	cachedData, found := cache.Get(cfg.previousUrl)
	if found {
		fmt.Println("Using cached data for:", cfg.previousUrl) //optional logging
		body = cachedData
	} else {

		//not in cache make HTTP request
		fmt.Println("Fetching new data for:", cfg.previousUrl) //optional logging
		res, err := http.Get(cfg.previousUrl)
		if err != nil {
			log.Fatal(err)
		}

		body, err = io.ReadAll(res.Body)
		defer res.Body.Close()
		if res.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
		}

		if err != nil {
			log.Fatal(err)
		}
		// Add to cache
		cache.Add(cfg.previousUrl, body)
	}

	var locations responseBody
	if err = json.Unmarshal(body, &locations); err != nil {
		return err
	}

	for _, location := range locations.Results {
		fmt.Println(location.Name)
	}

	cfg.nextUrl = locations.Next
	if locations.Previous != nil {
		cfg.previousUrl = locations.Previous.(string)
	} else {
		cfg.previousUrl = ""
	}

	return nil
}

func commandExplore(cfg *config, cache *pokecache.Cache, args ...string) error {
	//check if area is provided
	if len(args) == 0 {
		return fmt.Errorf("missing location area name or id")
	}

	areaName := args[0]
	fmt.Printf("Exploring %s...\n", areaName)

	//use the cache or API
	var body []byte
	var err error

	detailsUrl := "https://pokeapi.co/api/v2/location-area/" + areaName + "/"

	cachedData, found := cache.Get(detailsUrl)
	if found {
		fmt.Println("Using cached data for:", detailsUrl) //optional logging
		body = cachedData
	} else {

		//not in cache make HTTP request
		fmt.Println("Fetching new data for:", detailsUrl) //optional logging
		res, err := http.Get(detailsUrl)
		if err != nil {
			log.Fatal(err)
		}

		body, err = io.ReadAll(res.Body)
		defer res.Body.Close()
		if res.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
		}

		if err != nil {
			log.Fatal(err)
		}
		// Add to cache
		cache.Add(detailsUrl, body)
	}

	//dump response into locationDetails struct
	var locationData *locationDetails = &locationDetails{}
	if err = json.Unmarshal(body, locationData); err != nil {
		return err
	}

	//Output a list of found Pokemon
	if len(locationData.PokemonEncounters) == 0 {
		fmt.Println("No Pokemon found in this area.")
	} else {
		fmt.Println("Found Pokemon:")
		for _, encounter := range locationData.PokemonEncounters {
			fmt.Printf(" - %s\n", encounter.Pokemon.Name)
		}
	}

	return nil
}

func commandCatch(cfg *config, cache *pokecache.Cache, args ...string) error {

	//check if area is provided
	if len(args) == 0 {
		return fmt.Errorf("missing pokemon name or id")
	}

	pokemon := args[0]
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemon)

	//use the cache or API
	var body []byte
	var err error

	detailsUrl := "https://pokeapi.co/api/v2/pokemon/" + pokemon + "/"

	cachedData, found := cache.Get(detailsUrl)
	if found {
		fmt.Println("Using cached data for:", detailsUrl) //optional logging
		body = cachedData
	} else {

		//not in cache make HTTP request
		fmt.Println("Fetching new data for:", detailsUrl) //optional logging
		res, err := http.Get(detailsUrl)
		if err != nil {
			log.Fatal(err)
		}

		body, err = io.ReadAll(res.Body)
		defer res.Body.Close()
		if res.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
		}

		if err != nil {
			log.Fatal(err)
		}
		// Add to cache
		cache.Add(detailsUrl, body)
	}

	//dump response into locationDetails struct
	var pokemonData *pokemonDetails = &pokemonDetails{}
	if err = json.Unmarshal(body, pokemonData); err != nil {
		return err
	}

	//Determine if you actually caught a pokemon based on its base experience and random chance
	//Assuming 400 is around the max experience
	//Note this is a bit crap, but I can't be bothered to optimize your pokemon catching experience

	catchRoll := rand.Intn(400) + 50
	catchLimit := pokemonData.BaseExperience / 2

	if catchRoll >= catchLimit {
		fmt.Printf("%s was caught!\n", pokemon)
		//add pokemon to pokedex
		pokedex[pokemon] = *pokemonData
	} else {
		fmt.Printf("%s escaped!\n", pokemon)
	}

	return nil
}

func commandInspect(cfg *config, cache *pokecache.Cache, args ...string) error {

	pokemonName := args[0]

	pokemon, ok := pokedex[pokemonName]
	if !ok {
		fmt.Printf("you have not caught that pokemon")
	} else {
		fmt.Printf("Name: %s\n", pokemon.Name)
		fmt.Printf("Height: %d\n", pokemon.Height)
		fmt.Printf("Weight: %d\n", pokemon.Weight)
		fmt.Printf("Stats:\n")
		fmt.Printf("	-hp: %d\n", pokemon.Stats[0].BaseStat)
		fmt.Printf("	-attack: %d\n", pokemon.Stats[1].BaseStat)
		fmt.Printf("	-defense: %d\n", pokemon.Stats[2].BaseStat)
		fmt.Printf("	-special-attack: %d\n", pokemon.Stats[3].BaseStat)
		fmt.Printf("	-special-defense: %d\n", pokemon.Stats[4].BaseStat)
		fmt.Printf("	-speed: %d\n", pokemon.Stats[5].BaseStat)
		fmt.Printf("Types:\n")
		for _, pType := range pokemon.Types {
			fmt.Printf("	- %s\n", pType.Type.Name)
		}

	}

	return nil
}

func commandPokedex(cfg *config, cache *pokecache.Cache, args ...string) error {

	for _, pokemon := range pokedex {
		fmt.Printf(" - %s\n", pokemon.Name)
	}

	return nil
}

type cliCommand struct {
	name        string
	description string
	callback    func(cfg *config, cache *pokecache.Cache, args ...string) error
}

type config struct {
	previousUrl string
	nextUrl     string
}

type responseBody struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous any    `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

type locationDetails struct {
	EncounterMethodRates []struct {
		EncounterMethod struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"encounter_method"`
		VersionDetails []struct {
			Rate    int `json:"rate"`
			Version struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"version"`
		} `json:"version_details"`
	} `json:"encounter_method_rates"`
	GameIndex int `json:"game_index"`
	ID        int `json:"id"`
	Location  struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"location"`
	Name  string `json:"name"`
	Names []struct {
		Language struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"language"`
		Name string `json:"name"`
	} `json:"names"`
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"pokemon"`
		VersionDetails []struct {
			EncounterDetails []struct {
				Chance          int   `json:"chance"`
				ConditionValues []any `json:"condition_values"`
				MaxLevel        int   `json:"max_level"`
				Method          struct {
					Name string `json:"name"`
					URL  string `json:"url"`
				} `json:"method"`
				MinLevel int `json:"min_level"`
			} `json:"encounter_details"`
			MaxChance int `json:"max_chance"`
			Version   struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"version"`
		} `json:"version_details"`
	} `json:"pokemon_encounters"`
}

type pokemonDetails struct {
	Abilities []struct {
		Ability struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"ability"`
		IsHidden bool `json:"is_hidden"`
		Slot     int  `json:"slot"`
	} `json:"abilities"`
	BaseExperience int `json:"base_experience"`
	Cries          struct {
		Latest string `json:"latest"`
		Legacy string `json:"legacy"`
	} `json:"cries"`
	Forms []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"forms"`
	GameIndices []struct {
		GameIndex int `json:"game_index"`
		Version   struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"version"`
	} `json:"game_indices"`
	Height    int `json:"height"`
	HeldItems []struct {
		Item struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"item"`
		VersionDetails []struct {
			Rarity  int `json:"rarity"`
			Version struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"version"`
		} `json:"version_details"`
	} `json:"held_items"`
	ID                     int    `json:"id"`
	IsDefault              bool   `json:"is_default"`
	LocationAreaEncounters string `json:"location_area_encounters"`
	Moves                  []struct {
		Move struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"move"`
		VersionGroupDetails []struct {
			LevelLearnedAt  int `json:"level_learned_at"`
			MoveLearnMethod struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"move_learn_method"`
			Order        any `json:"order"`
			VersionGroup struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"version_group"`
		} `json:"version_group_details"`
	} `json:"moves"`
	Name          string `json:"name"`
	Order         int    `json:"order"`
	PastAbilities []struct {
		Abilities []struct {
			Ability  any  `json:"ability"`
			IsHidden bool `json:"is_hidden"`
			Slot     int  `json:"slot"`
		} `json:"abilities"`
		Generation struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"generation"`
	} `json:"past_abilities"`
	PastTypes []any `json:"past_types"`
	Species   struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"species"`
	Sprites struct {
		BackDefault      string `json:"back_default"`
		BackFemale       string `json:"back_female"`
		BackShiny        string `json:"back_shiny"`
		BackShinyFemale  string `json:"back_shiny_female"`
		FrontDefault     string `json:"front_default"`
		FrontFemale      string `json:"front_female"`
		FrontShiny       string `json:"front_shiny"`
		FrontShinyFemale string `json:"front_shiny_female"`
		Other            struct {
			DreamWorld struct {
				FrontDefault string `json:"front_default"`
				FrontFemale  any    `json:"front_female"`
			} `json:"dream_world"`
			Home struct {
				FrontDefault     string `json:"front_default"`
				FrontFemale      string `json:"front_female"`
				FrontShiny       string `json:"front_shiny"`
				FrontShinyFemale string `json:"front_shiny_female"`
			} `json:"home"`
			OfficialArtwork struct {
				FrontDefault string `json:"front_default"`
				FrontShiny   string `json:"front_shiny"`
			} `json:"official-artwork"`
			Showdown struct {
				BackDefault      string `json:"back_default"`
				BackFemale       string `json:"back_female"`
				BackShiny        string `json:"back_shiny"`
				BackShinyFemale  any    `json:"back_shiny_female"`
				FrontDefault     string `json:"front_default"`
				FrontFemale      string `json:"front_female"`
				FrontShiny       string `json:"front_shiny"`
				FrontShinyFemale string `json:"front_shiny_female"`
			} `json:"showdown"`
		} `json:"other"`
		Versions struct {
			GenerationI struct {
				RedBlue struct {
					BackDefault      string `json:"back_default"`
					BackGray         string `json:"back_gray"`
					BackTransparent  string `json:"back_transparent"`
					FrontDefault     string `json:"front_default"`
					FrontGray        string `json:"front_gray"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"red-blue"`
				Yellow struct {
					BackDefault      string `json:"back_default"`
					BackGray         string `json:"back_gray"`
					BackTransparent  string `json:"back_transparent"`
					FrontDefault     string `json:"front_default"`
					FrontGray        string `json:"front_gray"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"yellow"`
			} `json:"generation-i"`
			GenerationIi struct {
				Crystal struct {
					BackDefault           string `json:"back_default"`
					BackShiny             string `json:"back_shiny"`
					BackShinyTransparent  string `json:"back_shiny_transparent"`
					BackTransparent       string `json:"back_transparent"`
					FrontDefault          string `json:"front_default"`
					FrontShiny            string `json:"front_shiny"`
					FrontShinyTransparent string `json:"front_shiny_transparent"`
					FrontTransparent      string `json:"front_transparent"`
				} `json:"crystal"`
				Gold struct {
					BackDefault      string `json:"back_default"`
					BackShiny        string `json:"back_shiny"`
					FrontDefault     string `json:"front_default"`
					FrontShiny       string `json:"front_shiny"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"gold"`
				Silver struct {
					BackDefault      string `json:"back_default"`
					BackShiny        string `json:"back_shiny"`
					FrontDefault     string `json:"front_default"`
					FrontShiny       string `json:"front_shiny"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"silver"`
			} `json:"generation-ii"`
			GenerationIii struct {
				Emerald struct {
					FrontDefault string `json:"front_default"`
					FrontShiny   string `json:"front_shiny"`
				} `json:"emerald"`
				FireredLeafgreen struct {
					BackDefault  string `json:"back_default"`
					BackShiny    string `json:"back_shiny"`
					FrontDefault string `json:"front_default"`
					FrontShiny   string `json:"front_shiny"`
				} `json:"firered-leafgreen"`
				RubySapphire struct {
					BackDefault  string `json:"back_default"`
					BackShiny    string `json:"back_shiny"`
					FrontDefault string `json:"front_default"`
					FrontShiny   string `json:"front_shiny"`
				} `json:"ruby-sapphire"`
			} `json:"generation-iii"`
			GenerationIv struct {
				DiamondPearl struct {
					BackDefault      string `json:"back_default"`
					BackFemale       string `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  string `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      string `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale string `json:"front_shiny_female"`
				} `json:"diamond-pearl"`
				HeartgoldSoulsilver struct {
					BackDefault      string `json:"back_default"`
					BackFemale       string `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  string `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      string `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale string `json:"front_shiny_female"`
				} `json:"heartgold-soulsilver"`
				Platinum struct {
					BackDefault      string `json:"back_default"`
					BackFemale       string `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  string `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      string `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale string `json:"front_shiny_female"`
				} `json:"platinum"`
			} `json:"generation-iv"`
			GenerationV struct {
				BlackWhite struct {
					Animated struct {
						BackDefault      string `json:"back_default"`
						BackFemale       string `json:"back_female"`
						BackShiny        string `json:"back_shiny"`
						BackShinyFemale  string `json:"back_shiny_female"`
						FrontDefault     string `json:"front_default"`
						FrontFemale      string `json:"front_female"`
						FrontShiny       string `json:"front_shiny"`
						FrontShinyFemale string `json:"front_shiny_female"`
					} `json:"animated"`
					BackDefault      string `json:"back_default"`
					BackFemale       string `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  string `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      string `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale string `json:"front_shiny_female"`
				} `json:"black-white"`
			} `json:"generation-v"`
			GenerationVi struct {
				OmegarubyAlphasapphire struct {
					FrontDefault     string `json:"front_default"`
					FrontFemale      string `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale string `json:"front_shiny_female"`
				} `json:"omegaruby-alphasapphire"`
				XY struct {
					FrontDefault     string `json:"front_default"`
					FrontFemale      string `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale string `json:"front_shiny_female"`
				} `json:"x-y"`
			} `json:"generation-vi"`
			GenerationVii struct {
				Icons struct {
					FrontDefault string `json:"front_default"`
					FrontFemale  any    `json:"front_female"`
				} `json:"icons"`
				UltraSunUltraMoon struct {
					FrontDefault     string `json:"front_default"`
					FrontFemale      string `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale string `json:"front_shiny_female"`
				} `json:"ultra-sun-ultra-moon"`
			} `json:"generation-vii"`
			GenerationViii struct {
				Icons struct {
					FrontDefault string `json:"front_default"`
					FrontFemale  string `json:"front_female"`
				} `json:"icons"`
			} `json:"generation-viii"`
		} `json:"versions"`
	} `json:"sprites"`
	Stats []struct {
		BaseStat int `json:"base_stat"`
		Effort   int `json:"effort"`
		Stat     struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Slot int `json:"slot"`
		Type struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"type"`
	} `json:"types"`
	Weight int `json:"weight"`
}
