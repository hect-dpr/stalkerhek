package stalker

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
)

// Channel stores information about channel in Stalker portal. This is not a real TV channel representation, but details on how to retrieve a working channel's URL.
type Channel struct {
	cmd     string             // channel's identifier in Stalker portal
	logo    string             // Full URL to logo in Stalker portal
	portal  *Portal            // Reference to portal from where this channel is taken from
	genreID string             // Stores genre ID (category ID)
	genres  *map[string]string // Stores mappings for genre ID -> genre title
}

// NewLink retrieves a link to the working channel. Retrieved link can be played in VLC or Kodi, but expires very soon if not being constantly opened (used).
func (c *Channel) NewLink() (string, error) {
	type tmpStruct struct {
		Js struct {
			Cmd string `json:"cmd"`
		} `json:"js"`
	}
	var tmp tmpStruct

	link := c.portal.Location + "?action=create_link&type=itv&forced_storage=undefined&download=0&cmd=" + url.PathEscape(c.cmd) + "&JsHttpRequest=1-xml"
	content, err := c.portal.httpRequest(link)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(content, &tmp); err != nil {
		return "", err
	}

	strs := strings.Split(tmp.Js.Cmd, " ")
	if len(strs) == 2 {
		u, err := url.Parse(strs[1])
		if err != nil {
			panic(err)
		}

		if strings.Contains(c.cmd, "localhost") {
			return strs[1], nil
		} else {
			return strings.Split(c.cmd, " ")[1] + "?" + u.RawQuery, nil
		}

	}
	return "", errors.New("Stalker portal returned invalid link to TV Channel: " + tmp.Js.Cmd)
}

// Logo returns full link to channel's logo
func (c *Channel) Logo() string {
	if c.logo == "" {
		return ""
	}
	return c.portal.Location + "misc/logos/320/" + c.logo // hardcoded path - fixme?
}

// Genre returns a genre title
func (c *Channel) Genre() string {
	g, ok := (*c.genres)[c.genreID]
	if !ok {
		g = "Other"
	}
	return strings.Title(g)
}

// RetrieveChannels retrieves all TV channels from stalker portal.
func (p *Portal) RetrieveChannels() (map[string]*Channel, error) {
	type tmpStruct struct {
		Js struct {
			Data []struct {
				Name    string `json:"name"`
				Cmd     string `json:"cmd"`
				Logo    string `json:"logo"`
				GenreID string `json:"tv_genre_id"`
			} `json:"data"`
		} `json:"js"`
	}
	var tmp tmpStruct

	profile, err := p.httpRequest(p.Location + "?type=stb&action=get_profile&JsHttpRequest=1-xml")
	if err != nil {
		return nil, err
	}

	log.Println(profile)

	content, err := p.httpRequest(p.Location + "?action=get_all_channels&type=itv&&JsHttpRequest=1-xml")
	if err != nil {
		return nil, err
	}

	ioutil.WriteFile("/tmp/stalkerchannels.json", content, 0644)

	if err := json.Unmarshal(content, &tmp); err != nil {
		log.Println(string(content))
		panic(err)
	}

	genres, err := p.getGenres()
	if err != nil {
		return nil, err
	}

	log.Println(genres)

	// Build channels list and return
	channels := make(map[string]*Channel, len(tmp.Js.Data))
	for _, v := range tmp.Js.Data {
		channels[v.Name] = &Channel{
			cmd:     v.Cmd,
			logo:    v.Logo,
			portal:  p,
			genreID: v.GenreID,
			genres:  &genres,
		}
	}

	log.Println(channels)

	return channels, nil
}

func (p *Portal) getGenres() (map[string]string, error) {
	type tmpStruct struct {
		Js []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"js"`
	}
	var tmp tmpStruct

	content, err := p.httpRequest(p.Location + "?action=get_genres&type=itv&JsHttpRequest=1-xml")
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(content, &tmp); err != nil {
		log.Fatalln(string(content))
	}

	genres := make(map[string]string, len(tmp.Js))
	for _, el := range tmp.Js {
		genres[el.ID] = el.Title
	}

	return genres, nil
}
