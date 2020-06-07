package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/zorchenhimer/MovieNight/common"
)

const emoteDir = "./static/emotes/"

type TwitchUser struct {
	ID    string
	Login string
}

type EmoteInfo struct {
	ID   int
	Code string
}

func loadEmotes() error {
	//fmt.Println(processEmoteDir(emoteDir))
	fmt.Printf("Loading emotes from %v", emoteDir)
	newEmotes, err := processEmoteDir(emoteDir)
	if err != nil {
		return err
	}

	common.Emotes = newEmotes

	return nil
}

func processEmoteDir(path string) (common.EmotesMap, error) {
	dirInfo, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not open emoteDir:")
	}

	em := common.NewEmotesMap()

	for _, item := range dirInfo {
		// Get first level subdirs (eg, "twitch", "discord", etc)
		if item.IsDir() {
			fmt.Printf("Found top-level dir %v", item.Name())
			em[item.Name()] = map[string]map[string]string{}
			continue
		}
	}

	fmt.Sprintf("Emote tree %v", em)

	// Get second level subdirs (eg, "twitch", "zorchenhimer", etc)
	for dir, _ := range em {
		fmt.Printf("Looking in %v", dir)
		subd, err := ioutil.ReadDir(filepath.Join(path, dir))
		if err != nil {
			fmt.Printf("Error reading dir %q: %v\n", subd, err)
			continue
		}
		for _, d := range subd {
			fmt.Printf("Moving into %v/%v...", dir, subd)
			if d.IsDir() {
				p := filepath.Join(path, dir, d.Name())
				em[dir][d.Name()], err = findEmotes(p)
				if err != nil {
					fmt.Printf("Error finding emotes in %q: %v\n", p, err)
				}
			}
		}
	}

	fmt.Printf("processEmoteDir: %d\n", len(em))
	return em, nil
}

func findEmotes(dir string) (map[string]string, error) {
	fmt.Printf("finding emotes in %q\n", dir)
	emotePNGs, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		return em, fmt.Errorf("unable to glob emote directory: %s\n", err)
	}
	fmt.Printf("%d emotePNGs\n", len(emotePNGs))

	emoteGIFs, err := filepath.Glob(filepath.Join(dir, "*.gif"))
	if err != nil {
		return em, errors.Wrap(err, "unable to glob emote directory:")
	}
	fmt.Printf("%d emoteGIFs\n", len(emoteGIFs))

	em := map[string]string{}

	for _, file := range emotePNGs {
		code, filepath = parseEmoteName(file)
		em[code] = filepath
	}

	for _, file := range emoteGIFs {
		code, filepath = parseEmoteName(file)
		em[code] = filepath
	}

	return em, nil
}

func parseEmoteName(file string) (string, string) {
	fullpath = reStripStatic.ReplaceAllLiteralString(file, "")

	base := filepath.Base(fullpath)
	code := base[0 : len(base)-len(filepath.Ext(base))]

	return code, fullpath
}

func getEmotes(names []string) error {
	users := getUserIDs(names)
	users = append(users, TwitchUser{ID: "0", Login: "twitch"})

	for _, user := range users {
		emotes, cheers, err := getChannelEmotes(user.ID)
		if err != nil {
			return errors.Wrapf(err, "could not get emote data for \"%s\"", user.ID)
		}

		emoteUserDir := filepath.Join(emoteDir, "twitch", user.Login)
		if _, err := os.Stat(emoteUserDir); os.IsNotExist(err) {
			os.MkdirAll(emoteUserDir, os.ModePerm)
		}

		for _, emote := range emotes {
			if !strings.ContainsAny(emote.Code, `:;\[]|?&`) {
				filePath := filepath.Join(emoteUserDir, emote.Code+".png")
				file, err := os.Create(filePath)
				if err != nil {

					return errors.Wrapf(err, "could not create emote file in path \"%s\":", filePath)
				}

				err = downloadEmote(emote.ID, file)
				if err != nil {
					return errors.Wrapf(err, "could not download emote %s:", emote.Code)
				}
			}
		}

		for amount, sizes := range cheers {
			name := fmt.Sprintf("%sCheer%s.gif", user.Login, amount)
			filePath := filepath.Join(emoteUserDir, name)
			file, err := os.Create(filePath)
			if err != nil {
				return errors.Wrapf(err, "could not create emote file in path \"%s\":", filePath)
			}

			err = downloadCheerEmote(sizes["4"], file)
			if err != nil {
				return errors.Wrapf(err, "could not download emote %s:", name)
			}
		}
	}
	return nil
}

func getUserIDs(names []string) []TwitchUser {
	logins := strings.Join(names, "&login=")
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", logins), nil)
	if err != nil {
		log.Fatalln("Error generating new request:", err)
	}
	request.Header.Set("Client-ID", settings.TwitchClientID)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.TwitchClientSecret))

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatalln("Error sending request:", err)
	}

	decoder := json.NewDecoder(resp.Body)
	type userResponse struct {
		Data []TwitchUser
	}
	var data userResponse

	err = decoder.Decode(&data)
	if err != nil {
		log.Fatalln("Error decoding data:", err)
	}

	return data.Data
}

func getChannelEmotes(ID string) ([]EmoteInfo, map[string]map[string]string, error) {
	resp, err := http.Get("https://api.twitchemotes.com/api/v4/channels/" + ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get emotes")
	}
	decoder := json.NewDecoder(resp.Body)

	type EmoteResponse struct {
		Emotes     []EmoteInfo
		Cheermotes map[string]map[string]string
	}
	var data EmoteResponse

	err = decoder.Decode(&data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not decode emotes")
	}

	return data.Emotes, data.Cheermotes, nil
}

func downloadEmote(ID int, file *os.File) error {
	resp, err := http.Get(fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%d/3.0", ID))
	if err != nil {
		return errors.Errorf("could not download emote file %s: %v", file.Name(), err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Errorf("could not save emote: %v", err)
	}
	return nil
}

func downloadCheerEmote(url string, file *os.File) error {
	resp, err := http.Get(url)
	if err != nil {
		return errors.Errorf("could not download cheer file %s: %v", file.Name(), err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Errorf("could not save cheer: %v", err)
	}
	return nil
}
