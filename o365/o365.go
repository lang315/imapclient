// Package o365 implements an imap client, using Office 365 Mail REST API.
package o365

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/pkg/errors"
	"github.com/tgulacsi/oauth2client"
)

var Log = func(keyvals ...interface{}) error {
	log.Println(keyvals...)
	return nil
}

const baseURL = "https://outlook.office.com/api/v2.0/me"

type client struct {
	*oauth2.Config
	oauth2.TokenSource
}

func NewClient(clientID, clientSecret, redirectURL string) *client {
	if redirectURL == "" {
		redirectURL = "http://localhost:8123"
	}
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://outlook.office.com/mail.read"},
		Endpoint:     oauth2client.AzureV2Endpoint,
	}

	return &client{
		Config: conf,
		TokenSource: oauth2client.NewTokenSource(
			conf,
			filepath.Join(os.Getenv("HOME"), ".config", "o365.conf")),
	}
}

type Attachment struct {
	// The MIME type of the attachment.
	ContentType string `json:",omitempty"`
	// true if the attachment is an inline attachment; otherwise, false.
	IsInline bool `json:",omitempty"`
	// The date and time when the attachment was last modified. The date and time use ISO 8601 format and is always in UTC time. For example, midnight UTC on Jan 1, 2014 would look like this: '2014-01-01T00:00:00Z'
	LastModifiedDateTime time.Time `json:",omitempty"`
	// The display name of the attachment. This does not need to be the actual file name.
	Name string `json:",omitempty"`
	// The length of the attachment in bytes.
	Size int32 `json:",omitempty"`
}

type Recipient struct {
	EmailAddress EmailAddress `json:",omitempty"`
}

type EmailAddress struct {
	Name, Address string `json:",omitempty"`
}

type ItemBody struct {
	// The content type: Text = 0, HTML = 1.
	ContentType string `json:",omitempty"`
	// The text or HTML content.
	Content string `json:",omitempty"`
}

type Importance string
type InferenceClassificationType string
type SingleValueLegacyExtendedProperty struct {
	// A property values.
	Value string `json:",omitempty"`
	// The property ID. This is used to identify the property.
	PropertyID string `json:"PropertyId,omitempty"`
}
type MultiValueLegacyExtendedProperty struct {
	// A collection of property values.
	Value []string `json:",omitempty"`
	// The property ID. This is used to identify the property.
	PropertyID string `json:"PropertyId,omitempty"`
}

// https://msdn.microsoft.com/en-us/office/office365/api/complex-types-for-mail-contacts-calendar#MessageResource
// The fields last word designates the Writable/Filterable/Searchable property of the field.
type Message struct {
	// The FileAttachment and ItemAttachment attachments for the message. Navigation property.
	// W-S
	Attachments []Attachment `json:",omitempty"`
	// The Bcc recipients for the message.
	// W-S
	BccRecipients []Recipient `json:",omitempty"`
	// The body of the message.
	// W--
	Body ItemBody `json:",omitempty"`
	// The first 255 characters of the message body content.
	// --S
	BodyPreview string `json:",omitempty"`
	// The categories associated with the message.
	// WFS
	Categories []string `json:",omitempty"`
	// The Cc recipients for the message.
	// W-S
	CcRecipients []Recipient `json:",omitempty"`
	// The version of the message.
	// ---
	ChangeKey string `json:",omitempty"`
	// The ID of the conversation the email belongs to.
	// -F-
	ConversationID string `json:"ConversationId,omitempty"`
	// The date and time the message was created.
	// -F-
	CreatedDateTime *time.Time `json:",omitempty"`
	// The collection of open type data extensions defined for the message. Navigation property.
	// -F-
	Extensions []string `json:",omitempty"`
	// The mailbox owner and sender of the message.
	// WFS
	From *Recipient `json:",omitempty"`
	// Indicates whether the message has attachments.
	// -FS
	HasAttachments bool `json:",omitempty"`
	// The unique identifier of the message.
	// ---
	ID string `json:"Id,omitempty"`
	// The importance of the message: Low = 0, Normal = 1, High = 2.
	// WFS
	Importance Importance `json:",omitempty"`
	// The classification of this message for the user, based on inferred relevance or importance, or on an explicit override.
	// WFS
	InferenceClassification InferenceClassificationType `json:",omitempty"`
	// Indicates whether a read receipt is requested for the message.
	// WF-
	IsDeliveryReceiptRequested bool `json:",omitempty"`
	// Indicates whether the message is a draft. A message is a draft if it hasn't been sent yet.
	// -F-
	IsDraft bool `json:",omitempty"`
	// Indicates whether the message has been read.
	// WF-
	IsRead bool `json:",omitempty"`
	// Indicates whether a read receipt is requested for the message.
	// WF-
	IsReadReceiptRequested bool `json:",omitempty"`
	// The date and time the message was last changed.
	// -F-
	LastModifiedDateTime *time.Time `json:",omitempty"`
	// A collection of multi-value extended properties of type MultiValueLegacyExtendedProperty. This is a navigation property. Find more information about extended properties.
	// WF-
	MultiValueExtendedProperties *MultiValueLegacyExtendedProperty `json:",omitempty"`
	// The unique identifier for the message's parent folder.
	// ---
	ParentFolderID string `json:"ParentFolderId,omitempty"`
	// The date and time the message was received.
	// -FS
	ReceivedDateTime *time.Time `json:",omitempty"`
	// The email addresses to use when replying.
	// ---
	ReplyTo []Recipient `json:",omitempty"`
	// The account that is actually used to generate the message.
	// WF-
	Sender *Recipient `json:",omitempty"`
	// A collection of single-value extended properties of type SingleValueLegacyExtendedProperty. This is a navigation property. Find more information about extended properties.
	// WF-
	SingleValueExtendedProperties *SingleValueLegacyExtendedProperty `json:",omitempty"`
	// The date and time the message was sent.
	// -F-
	SentDateTime *time.Time `json:",omitempty"`
	// The subject of the message.
	// WF-
	Subject string `json:",omitempty"`
	// The To recipients for the message.
	// W-S
	ToRecipients []Recipient `json:",omitempty"`
	// The body of the message that is unique to the conversation.
	// ---
	UniqueBody *ItemBody `json:",omitempty"`
	// The URL to open the message in Outlook Web App.
	// You can append an ispopout argument to the end of the URL to change how the message is displayed. If ispopout is not present or if it is set to 1, then the message is shown in a popout window. If ispopout is set to 0, then the browser will show the message in the Outlook Web App review pane.
	// The message will open in the browser if you are logged in to your mailbox via Outlook Web App. You will be prompted to login if you are not already logged in with the browser.
	// This URL can be accessed from within an iFrame.
	// -F-
	WebLink string `json:",omitempty"`
}

func (c *client) List(ctx context.Context, mbox, pattern string, all bool) ([]Message, error) {
	path := "/messages"
	if mbox != "" {
		path = "/MailFolders/" + mbox + "/messages"
	}

	values := url.Values{
		"$select": {"Sender,Subject"},
	}
	if pattern != "" {
		values.Set("$search", `"subject:`+pattern+`"`)
	}
	if !all {
		values.Set("$filter", "IsRead eq false")
	}

	body, err := c.get(ctx, path+"?"+values.Encode())
	if err != nil {
		return nil, err
	}
	defer body.Close()

	type listResponse struct {
		Value []Message `json:"value"`
	}
	var resp listResponse
	err = json.NewDecoder(body).Decode(&resp)
	return resp.Value, err
}

func (c *client) Get(ctx context.Context, msgID string) (Message, error) {
	path := "/messages/" + msgID
	var msg Message
	body, err := c.get(ctx, path)
	if err != nil {
		return msg, err
	}
	defer body.Close()
	err = json.NewDecoder(body).Decode(&msg)
	return msg, err
}

func (c *client) Send(ctx context.Context, msg Message) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(struct {
		Message Message
	}{Message: msg}); err != nil {
		return errors.Wrapf(err, "encode %#v", msg)
	}
	path := "/sendmail"
	return c.post(ctx, path, bytes.NewReader(buf.Bytes()))
}

func (c *client) post(ctx context.Context, path string, body io.Reader) error {
	var buf bytes.Buffer
	resp, err := oauth2.NewClient(ctx, c.TokenSource).
		Post(baseURL+path, "application/json", io.TeeReader(body, &buf))
	if err != nil {
		return errors.Wrap(err, buf.String())
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		io.Copy(&buf, body)
		io.WriteString(&buf, "\n\n")
		io.Copy(&buf, resp.Body)
		return errors.Errorf("POST %q: %s\n%s", path, resp.Status, buf.Bytes())
	}
	return nil
}

func (c *client) Delete(ctx context.Context, msgID string) error {
	return c.delete(ctx, "/messages/"+msgID)
}

func (c *client) Move(ctx context.Context, msgID, destinationID string) error {
	return c.post(ctx, "/messages/"+msgID+"/move", bytes.NewReader(jsonObj("DestinationId", destinationID)))
}
func (c *client) Copy(ctx context.Context, msgID, destinationID string) error {
	return c.post(ctx, "/messages/"+msgID+"/copy", bytes.NewReader(jsonObj("DestinationId", destinationID)))
}

func (c *client) CreateFolder(ctx context.Context, parent, folder string) error {
	return c.post(ctx, "/MailFolders/"+parent+"/childfolders", bytes.NewReader(jsonObj("DisplayName", folder)))
}

func (c *client) RenameFolder(ctx context.Context, folderID, newName string) error {
	return c.post(ctx, "/MailFolders/"+folderID, bytes.NewReader(jsonObj("DisplayName", newName)))
}
func (c *client) MoveFolder(ctx context.Context, folderID, destinationID string) error {
	return c.post(ctx, "/MailFolders/"+folderID+"/move", bytes.NewReader(jsonObj("DestinationId", destinationID)))
}
func (c *client) CopyFolder(ctx context.Context, folderID, destinationID string) error {
	return c.post(ctx, "/MailFolders/"+folderID+"/copy", bytes.NewReader(jsonObj("DestinationId", destinationID)))
}

func (c *client) DeleteFolder(ctx context.Context, folderID string) error {
	return c.delete(ctx, "/MailFolders/"+folderID)
}

func (c *client) get(ctx context.Context, path string) (io.ReadCloser, error) {
	Log("get", baseURL+path)
	resp, err := oauth2.NewClient(ctx, c.TokenSource).Get(baseURL + path)
	if err != nil {
		return nil, errors.Wrap(err, path)
	}
	return resp.Body, nil
}

func (c *client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequest("DELETE", baseURL+path, nil)
	if err != nil {
		return errors.Wrap(err, path)
	}
	resp, err := oauth2.NewClient(ctx, c.TokenSource).Do(req)
	if err != nil {
		return errors.Wrap(err, req.URL.String())
	}
	if resp.StatusCode > 299 {
		return errors.Errorf("DELETE %q: %s", path, resp.Status)
	}
	return nil
}

func jsonObj(key, value string) []byte {
	b, err := json.Marshal(map[string]string{key: value})
	if err != nil {
		panic(err)
	}
	return b
}