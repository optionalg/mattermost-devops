package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/cpanato/onelogin"
)

type errorMsg struct {
	Error string `json:"error"`
}

func clientError(status int, message errorMsg) (events.APIGatewayProxyResponse, error) {
	jsonString, _ := json.Marshal(message)
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(jsonString),
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	msg := errorMsg{
		Error: err.Error(),
	}
	jsonString, _ := json.Marshal(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       string(jsonString),
	}, nil
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var oneLoginEvents []OneLogin
	if err := json.Unmarshal([]byte(req.Body), &oneLoginEvents); err != nil {
		return serverError(errors.New("Error decoding"))
	}

	for _, event := range oneLoginEvents {
		switch event.EventTypeID {
		case 13: // CREATED_USER
			fmt.Println("Create user event. Will start the onboarding")
			onBoardUser(event.UserID)
		// DEACTIVED_USER or SUSPENDED_USER or USER_UNLICENSED
		case 15, 21, 223:
			fmt.Println("Create user event. Will start the offboading")
			offBoardUser(event.UserID)
		default:
			fmt.Printf("Event not needed %v\n", event.EventTypeID)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string("{\"status\":\"ok\"}"),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}

func onBoardUser(userID int64) {
	c := onelogin.New(os.Getenv("ONELOGIN_CLIENT"), os.Getenv("ONELOGIN_CLIENTSECRET"), "us", os.Getenv("ONELOGIN_SUBDOMAIN"))

	onBoardUser, err := c.User.GetUser(context.Background(), userID)
	if err != nil {
		fmt.Printf("Error getting the user from onelogin err %v", err.Error())
		return
	}

	if onBoardUser.CustomAttributes["GH"] == "" {
		fmt.Printf("No GitHub handler for this user will not add.")
		return
	}

	userName := fmt.Sprintf("%s %s", onBoardUser.FirstName, onBoardUser.LastName)
loop:
	for _, role := range onBoardUser.RoleIDs {
		switch role {
		// Dev
		case 258872:
			fmt.Printf("Will add user to Developer Team in github - github handler: %s\n", onBoardUser.CustomAttributes["GH"])
			githubDev, _ := strconv.ParseInt(os.Getenv("GITHUB_DEV_TEAMID"), 10, 64)
			addUserToGithubTeam(githubDev, userName, onBoardUser.CustomAttributes["GH"])
			break loop
		// QA
		case 258878:
			fmt.Printf("Will add user to QA Team in github - github handler: %s\n", onBoardUser.CustomAttributes["GH"])
			githubQA, _ := strconv.ParseInt(os.Getenv("GITHUB_QA_TEAMID"), 10, 64)
			addUserToGithubTeam(githubQA, userName, onBoardUser.CustomAttributes["GH"])
			break loop
		// SA
		case 258880:
			fmt.Printf("Will add user to SA Team in github - github handler: %s\n", onBoardUser.CustomAttributes["GH"])
			githubSA, _ := strconv.ParseInt(os.Getenv("GITHUB_SA_TEAMID"), 10, 64)
			addUserToGithubTeam(githubSA, userName, onBoardUser.CustomAttributes["GH"])
			break loop
		// PM
		case 258875:
			fmt.Printf("Will add user to PM Team in github - github handler: %s\n", onBoardUser.CustomAttributes["GH"])
			githubPM, _ := strconv.ParseInt(os.Getenv("GITHUB_PM_TEAMID"), 10, 64)
			addUserToGithubTeam(githubPM, userName, onBoardUser.CustomAttributes["GH"])
			break loop
		default:
			fmt.Println("No roles match")
			continue
		}
	}
}

func offBoardUser(userID int64) {
	c := onelogin.New(os.Getenv("ONELOGIN_CLIENT"), os.Getenv("ONELOGIN_CLIENTSECRET"), "us", os.Getenv("ONELOGIN_SUBDOMAIN"))

	offBoardUser, err := c.User.GetUser(context.Background(), userID)
	if err != nil {
		fmt.Printf("Error getting the user from onelogin err %v", err.Error())
		return
	}

	if offBoardUser.CustomAttributes["GH"] == "" {
		fmt.Printf("No GitHub handler for this user will not remove.")
		return
	}

	userName := fmt.Sprintf("%s %s", offBoardUser.FirstName, offBoardUser.LastName)
loop:
	for _, role := range offBoardUser.RoleIDs {
		switch role {
		// Dev
		case 258872:
			fmt.Printf("Will remove user to Developer Team in github - github handler: %s\n", offBoardUser.CustomAttributes["GH"])
			githubDev, _ := strconv.ParseInt(os.Getenv("GITHUB_DEV_TEAMID"), 10, 64)
			removeUserToGithubTeam(githubDev, userName, offBoardUser.CustomAttributes["GH"])
			break loop
		// QA
		case 258878:
			fmt.Printf("Will remove user to QA Team in github - github handler: %s\n", offBoardUser.CustomAttributes["GH"])
			githubQA, _ := strconv.ParseInt(os.Getenv("GITHUB_QA_TEAMID"), 10, 64)
			removeUserToGithubTeam(githubQA, userName, offBoardUser.CustomAttributes["GH"])
			break loop
		// SA
		case 258880:
			fmt.Printf("Will remove user to SA Team in github - github handler: %s\n", offBoardUser.CustomAttributes["GH"])
			githubSA, _ := strconv.ParseInt(os.Getenv("GITHUB_SA_TEAMID"), 10, 64)
			removeUserToGithubTeam(githubSA, userName, offBoardUser.CustomAttributes["GH"])
			break loop
		// PM
		case 258875:
			fmt.Printf("Will remove user to PM Team in github - github handler: %s\n", offBoardUser.CustomAttributes["GH"])
			githubPM, _ := strconv.ParseInt(os.Getenv("GITHUB_PM_TEAMID"), 10, 64)
			removeUserToGithubTeam(githubPM, userName, offBoardUser.CustomAttributes["GH"])
			break loop
		default:
			fmt.Println("No roles match")
			continue
		}
	}
}

func addUserToGithubTeam(teamID int64, UserName, githubHandler string) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)
	_, _, err := client.Teams.AddTeamMembership(context.Background(), teamID, githubHandler, nil)
	if err != nil {
		fmt.Printf("Error adding the user to the team. Err=%v\n", err.Error())
	}

	notifyMattermost(fmt.Sprintf("User %s with github handler %s added to github team", UserName, githubHandler))
}

func removeUserToGithubTeam(teamID int64, UserName, githubHandler string) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)
	_, err := client.Teams.RemoveTeamMembership(context.Background(), teamID, githubHandler)
	if err != nil {
		fmt.Printf("Error removing the user to the team. Err=%v\n", err.Error())
	}

	notifyMattermost(fmt.Sprintf("User %s with github handler %s removed from github team", UserName, githubHandler))
}

type MMNotify struct {
	Text     string `json:text`
	Username string `json:username`
}

func notifyMattermost(msg string) {

	mm := MMNotify{
		Text:     msg,
		Username: "OnOffBoardBot",
	}

	b, err := json.Marshal(mm)
	req, err := http.NewRequest("POST", os.Getenv("MATTERMOST_HOOK"), bytes.NewBuffer(b))
	if err != nil {
		fmt.Println("Error creating the request", err.Error())
		return
	}
	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		fmt.Println("Error doing the post", err.Error())
		return
	}
}

type OneLogin struct {
	AccountID                       int    `json:"account_id,omitempty"`
	ActorSystem                     string `json:"actor_system,omitempty"`
	ActorUserID                     int    `json:"actor_user_id,omitempty"`
	ActorUserName                   string `json:"actor_user_name,omitempty"`
	AdcID                           int    `json:"adc_id,omitempty"`
	AdcName                         string `json:"adc_name,omitempty"`
	APICredentialName               string `json:"api_credential_name,omitempty"`
	AppID                           int    `json:"app_id,omitempty"`
	AppName                         string `json:"app_name,omitempty"`
	AssumedBySuperadminOrReseller   string `json:"assumed_by_superadmin_or_reseller,omitempty"`
	AssumingActingUserID            int    `json:"assuming_acting_user_id,omitempty"`
	AuthenticationFactorDescription string `json:"authentication_factor_description,omitempty"`
	AuthenticationFactorID          int    `json:"authentication_factor_id,omitempty"`
	AuthenticationFactorType        string `json:"authentication_factor_type,omitempty"`
	BrowserFingerprint              int    `json:"browser_fingerprint,omitempty"`
	CertificateID                   int    `json:"certificate_id,omitempty"`
	CertificateName                 string `json:"certificate_name,omitempty"`
	ClientID                        int    `json:"client_id,omitempty"`
	Create                          struct {
		ID string `json:"ID,omitempty"`
	} `json:"create,omitempty"`
	CustomMessage      string `json:"custom_message,omitempty"`
	DirectoryID        int    `json:"directory_id,omitempty"`
	DirectoryName      string `json:"directory_name,omitempty"`
	DirectorySyncRunID int    `json:"directory_sync_run_id,omitempty"`
	Entity             string `json:"entity,omitempty"`
	ErrorDescription   string `json:"error_description,omitempty"`
	EventTimestamp     string `json:"event_timestamp,omitempty"`
	EventTypeID        int    `json:"event_type_id,omitempty"`
	GroupID            int    `json:"group_id,omitempty"`
	GroupName          string `json:"group_name,omitempty"`
	ImportedUserID     int    `json:"imported_user_id,omitempty"`
	ImportedUserName   string `json:"imported_user_name,omitempty"`
	Ipaddr             string `json:"ipaddr,omitempty"`
	LoginID            int    `json:"login_id,omitempty"`
	LoginName          string `json:"login_name,omitempty"`
	MappingID          int    `json:"mapping_id,omitempty"`
	MappingName        string `json:"mapping_name,omitempty"`
	NoteID             int    `json:"note_id,omitempty"`
	NoteTitle          string `json:"note_title,omitempty"`
	Notes              string `json:"notes,omitempty"`
	ObjectID           int    `json:"object_id,omitempty"`
	OtpDeviceID        int    `json:"otp_device_id,omitempty"`
	OtpDeviceName      string `json:"otp_device_name,omitempty"`
	Param              string `json:"param,omitempty"`
	PolicyID           int    `json:"policy_id,omitempty"`
	PolicyName         string `json:"policy_name,omitempty"`
	PolicyType         string `json:"policy_type,omitempty"`
	PrivilegeID        int    `json:"privilege_id,omitempty"`
	PrivilegeName      string `json:"privilege_name,omitempty"`
	ProxyAgentID       int    `json:"proxy_agent_id,omitempty"`
	ProxyAgentName     string `json:"proxy_agent_name,omitempty"`
	ProxyIP            string `json:"proxy_ip,omitempty"`
	RadiusConfigID     int    `json:"radius_config_id,omitempty"`
	RadiusConfigName   string `json:"radius_config_name,omitempty"`
	Resolution         string `json:"resolution,omitempty"`
	ResolvedAt         string `json:"resolved_at,omitempty"`
	ResolvedByUserID   int    `json:"resolved_by_user_id,omitempty"`
	ResourceTypeID     int    `json:"resource_type_id,omitempty"`
	RiskCookieID       int    `json:"risk_cookie_id,omitempty"`
	RiskReasons        string `json:"risk_reasons,omitempty"`
	RiskScore          int    `json:"risk_score,omitempty"`
	RoleID             string `json:"role_id,omitempty"`
	RoleName           string `json:"role_name,omitempty"`
	ServiceDirectoryID string `json:"service_directory_id,omitempty"`
	Solved             string `json:"solved,omitempty"`
	TaskName           string `json:"task_name,omitempty"`
	TrustedIdpID       string `json:"trusted_idp_id,omitempty"`
	TrustedIdpName     string `json:"trusted_idp_name,omitempty"`
	UserAgent          string `json:"user_agent,omitempty"`
	UserFieldID        string `json:"user_field_id,omitempty"`
	UserFieldName      string `json:"user_field_name,omitempty"`
	UserID             int64  `json:"user_id,omitempty"`
	UserName           string `json:"user_name,omitempty"`
	UUID               string `json:"uuid,omitempty"`
}
