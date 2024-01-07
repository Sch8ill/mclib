package slp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Response represents the Server List Ping (SLP) response.
type Response struct {
	// Documentation link:
	// https://wiki.vg/Server_List_Ping
	Version            Version     `json:"version"`
	Players            Players     `json:"players"`
	Favicon            string      `json:"favicon,omitempty"`
	Description        Description `json:"description"`
	EnforcesSecureChat bool        `json:"enforcesSecureChat,omitempty"`
	PreviewsChat       bool        `json:"previewsChat,omitempty"`

	// Forge related data
	// https://wiki.vg/Minecraft_Forge_Handshake#Changes_to_Server_List_Ping
	ForgeModInfo *LegacyForgeModInfo `json:"modinfo,omitempty"`   // Minecraft Forge 1.7 - 1.12
	ForgeData    *ForgeData          `json:"forgeData,omitempty"` // Minecraft Forge 1.13 - Current

	// Latency measured by the client
	Latency int `json:"latency,omitempty"`
}

// Version represents the version information in the SLP response.
type Version struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

// Players represents player information in the SLP response.
type Players struct {
	Max    int      `json:"max"`
	Online int      `json:"online"`
	Sample []Player `json:"sample,omitempty"`
}

// Player represents an individual player's information in the SLP response.
type Player struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// ForgeData represents Forge mod data in the SLP response.
type ForgeData struct {
	Channels          []ForgeChannel `json:"channels"`
	Mods              []ForgeMod     `json:"mods"`
	FMLNetworkVersion int            `json:"fmlNetworkVersion"`
}

// ForgeChannel represents a Forge mod channel in ForgeData.
type ForgeChannel struct {
	Res      string `json:"res"`
	Version  string `json:"version"`
	Required bool   `json:"required"`
}

// ForgeMod represents a Forge mod in ForgeData.
type ForgeMod struct {
	ModID     string `json:"modId"`
	ModMarker string `json:"modmarker"`
}

// LegacyForgeModInfo represents legacy Forge mod information in the SLP response.
type LegacyForgeModInfo struct {
	Type    string           `json:"type"`
	ModList []LegacyForgeMod `json:"modList"`
}

// LegacyForgeMod represents a legacy Forge mod in LegacyForgeModInfo.
type LegacyForgeMod struct {
	ModID   string `json:"modid"`
	Version string `json:"version"`
}

// Description represents a Description in the SLP response.
// Description wraps a ChatComponent due to encoding limitations with dynamic JSON in go.
type Description struct {
	Description ChatComponent
}

// String converts the Description into a string.
func (d *Description) String() string {
	return d.Description.String()
}

// UnmarshalJSON unmarshalls a description into a ChatComponent.
// The description can be represented as a ChatComponent or string.
func (d *Description) UnmarshalJSON(b []byte) error {
	// ToDo: translate color/formatting codes to JSON
	// https://wiki.vg/Chat
	// https://github.com/Sch8ill/rcon/blob/master/color/color.go
	if b[0] == '"' {
		var text string
		if err := json.Unmarshal(b, &text); err != nil {
			return err
		}
		d.Description.Text = text

		return nil
	}

	if err := json.Unmarshal(b, &d.Description); err != nil {
		return err
	}

	return nil
}

// MarshalJSON marshals a Description by returning a marshalled ChatComponent.
func (d *Description) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Description)
}

// ChatComponent represents a Minecraft chat type used in the SLP response description.
type ChatComponent struct {
	Text          string        `json:"text"`
	Bold          bool          `json:"bold,omitempty"`
	Italic        bool          `json:"italic,omitempty"`
	Underlined    bool          `json:"underlined,omitempty"`
	Strikethrough bool          `json:"strikethrough,omitempty"`
	Obfuscated    bool          `json:"obfuscated,omitempty"`
	Font          string        `json:"font,omitempty"`
	Color         string        `json:"color,omitempty"`
	Insertion     string        `json:"insertion,omitempty"`
	ClickEvent    *ClickEvent   `json:"clickEvent,omitempty"`
	HoverEvent    *HoverEvent   `json:"hoverEvent,omitempty"`
	Extra         []Description `json:"extra,omitempty"`
}

// String converts the ChatComponent into a string.
func (c *ChatComponent) String() string {
	text := c.Text
	for _, extra := range c.Extra {
		text += extra.String()
	}

	return text
}

// ClickEvent represents click event inside a chat component.
type ClickEvent struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

// HoverEvent represents a hover event inside a chat component.
type HoverEvent struct {
	Action   string `json:"action"`
	Contents string `json:"contents"`
}

// NewResponse parses a raw SLP response string into a Response struct.
func NewResponse[T []byte | string](rawRes T) (*Response, error) {
	res := new(Response)
	if err := json.Unmarshal([]byte(rawRes), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// String converts the response to a JSON string.
func (r *Response) String() (string, error) {
	res, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to convert to JSON: %w", err)
	}

	return string(res), nil
}

// Icon decodes the favicon string into byte data.
func (r *Response) Icon() ([]byte, error) {
	if r.Favicon == "" {
		return nil, errors.New("status response does not contain a favicon")
	}

	base64Icon := strings.TrimPrefix(r.Favicon, "data:image/png;base64,")
	iconBytes, err := base64.StdEncoding.DecodeString(base64Icon)
	if err != nil {
		return nil, fmt.Errorf("failed to convert base64 image to bytes: %w", err)
	}

	return iconBytes, nil
}
