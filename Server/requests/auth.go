package spartanreport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var refreshTokenMap = make(map[string]RefreshTokenInfo)

// SetTokenInfo adds or updates token information in the map
func SetTokenInfo(token string, info RefreshTokenInfo) {
	refreshTokenMap[token] = info
}

// DeleteTokenInfo removes token information from the map
func DeleteTokenInfo(token string) {
	delete(refreshTokenMap, token)
}

// GetTokenInfo retrieves token information from the map
func GetTokenInfo(token string) (RefreshTokenInfo, bool) {
	info, exists := refreshTokenMap[token]
	return info, exists
}

func PrintOutTokenInfo() {
	for key, _ := range refreshTokenMap {
		fmt.Printf("Key: %s", key)
	}
}

type OAuthResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
}

type UserTokenRequest struct {
	Properties struct {
		AuthMethod string `json:"AuthMethod"`
		SiteName   string `json:"SiteName"`
		RpsTicket  string `json:"RpsTicket"`
	} `json:"Properties"`
	RelyingParty string `json:"RelyingParty"`
	TokenType    string `json:"TokenType"`
}

type UserTokenResponse struct {
	DisplayClaims struct {
		Xui []struct {
			Uhs string `json:"uhs"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
	Token string `json:"Token"`
}

type RefreshTokenInfo struct {
	RefreshToken    string    `json:"refresh_token"`
	ExpirationData  time.Time `json:"expiration_data"`
	OAuthExpiration time.Time `json:"oauth_expiration"`
}

// requestUserToken sends a request to get the user token and returns the response body.
func RequestUserToken(accessToken string) (*UserTokenResponse, error) {
	userTokenResp := &UserTokenResponse{}
	// Prepare the request payload
	reqData := UserTokenRequest{
		Properties: struct {
			AuthMethod string `json:"AuthMethod"`
			SiteName   string `json:"SiteName"`
			RpsTicket  string `json:"RpsTicket"`
		}{
			AuthMethod: "RPS",
			SiteName:   "user.auth.xboxlive.com",
			RpsTicket:  "d=" + accessToken,
		},
		RelyingParty: "http://auth.xboxlive.com",
		TokenType:    "JWT",
	}

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return userTokenResp, fmt.Errorf("Error marshaling JSON: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://user.auth.xboxlive.com/user/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		return userTokenResp, fmt.Errorf("Error creating request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-xbl-contract-version", "1")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return userTokenResp, fmt.Errorf("Error executing request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return userTokenResp, fmt.Errorf("Error reading response body: %v", err)
	}

	// Unmarshal the response body into the UserTokenResponse struct
	err = json.Unmarshal(body, &userTokenResp)
	if err != nil {
		return userTokenResp, fmt.Errorf("Error unmarshaling response: %v", err)
	}

	return userTokenResp, nil
}

// tempGamerInfoMap for temporary storage
var tempGamerInfoMap = make(map[string]GamerInfo)

// SetGamerInfo adds gamerInfo to the map
func SetGamerInfo(token string, gamerInfo GamerInfo) {
	tempGamerInfoMap[token] = gamerInfo
}

// GetGamerInfo retrieves and deletes gamerInfo from the map
func GetGamerInfo(token string) (GamerInfo, bool) {
	gamerInfo, exists := tempGamerInfoMap[token]
	if exists {
		delete(tempGamerInfoMap, token) // Delete after retrieval
	}
	return gamerInfo, exists
}

func ProcessAuthCode(code string, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received authorization code:", code)

	// Make the OAuth request
	// Load environment variables from .env file
	err := godotenv.Load("azure-keys.env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURI := os.Getenv("REDIRECT_URI")

	body := RequestOAuth(clientID, clientSecret, redirectURI, code)

	// Parse the OAuth response
	var oauthResp OAuthResponse
	err = json.Unmarshal(body, &oauthResp)

	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}
	// Request the user token
	userToken, err := RequestUserToken(oauthResp.AccessToken)
	if err != nil {
		fmt.Println("Error with AccessToken:", err)
		return
	}

	// Request the XSTS token
	err, SpartanResp := RequestXstsToken(*userToken)

	if err != nil {
		fmt.Println("Error with XSTS Token:", err)
		return
	}
	// Get the current time
	now := time.Now()
	expirationTime := now.Add(time.Second * time.Duration(oauthResp.ExpiresIn))

	tokenData := RefreshTokenInfo{
		RefreshToken:    oauthResp.RefreshToken,
		ExpirationData:  expirationTime,
		OAuthExpiration: expirationTime,
	}
	SetTokenInfo(SpartanResp.SpartanToken, tokenData)

	gamerInfo, err := RequestUserProfile(SpartanResp.SpartanToken)
	// Temporarily store gamerInfo with SpartanToken as the key
	SetGamerInfo(SpartanResp.SpartanToken, gamerInfo)
	// Send the SpartanToken to the client
	host := os.Getenv("HOST")
	redirectURL := fmt.Sprintf("%s/?token=%s", host, SpartanResp.SpartanToken)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)

}

func ProcessAuthCodeWithRefresh(code string, w http.ResponseWriter, r *http.Request, OAuthExpiration time.Time) (SpartanTokenResponse, error) {
	fmt.Println("Received authorization code:", code)
	SpartanTokenResponse := SpartanTokenResponse{}
	// Make the OAuth request
	// Load environment variables from .env file
	err := godotenv.Load("azure-keys.env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURI := os.Getenv("REDIRECT_URI")

	body, err := RequestOAuthWithRefreshToken(clientID, clientSecret, redirectURI, code)
	if err != nil {
		fmt.Println("Error with RefreshToken:", err)
		return SpartanTokenResponse, err
	}
	// Parse the OAuth response
	var oauthResp OAuthResponse
	err = json.Unmarshal(body, &oauthResp)

	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return SpartanTokenResponse, err
	}
	// Request the user token
	userToken, err := RequestUserToken(oauthResp.AccessToken)
	if err != nil {
		fmt.Println("Error with AccessToken:", err)
		return SpartanTokenResponse, err
	}

	// Request the XSTS token
	err, SpartanResp := RequestXstsToken(*userToken)

	if err != nil {
		fmt.Println("Error with XSTS Token:", err)
		return SpartanTokenResponse, err
	}

	now := time.Now()
	expirationTime := now.Add(time.Second * time.Duration(oauthResp.ExpiresIn))

	tokenData := RefreshTokenInfo{
		RefreshToken:    oauthResp.RefreshToken,
		ExpirationData:  expirationTime,
		OAuthExpiration: OAuthExpiration,
	}

	// Set the new token info in the map
	SetTokenInfo(SpartanResp.SpartanToken, tokenData)
	return SpartanResp, nil
}
func CheckAndUpdateGamerInfo(c *gin.Context, gamerInfoPassedIn GamerInfo) GamerInfo {
	spartanTokenPassedIn := gamerInfoPassedIn.SpartanKey
	// Check if the SpartanToken is in the map
	tokenInfo, exists := GetTokenInfo(spartanTokenPassedIn)
	if exists {
		fmt.Println("Token Expiration: ", tokenInfo.ExpirationData)
		currentTime := time.Now()
		expirationTime := tokenInfo.ExpirationData
		OAuthExpiration := tokenInfo.OAuthExpiration
		if OAuthExpiration.Before(currentTime) {
			fmt.Println("OAuth Token has expired")
			DeleteTokenInfo(spartanTokenPassedIn)
			return GamerInfo{XBLToken: "", SpartanKey: "", Gamertag: "", XUID: ""}
		}
		// Check if the token is expired or expiring within 5 minutes
		if expirationTime.Before(currentTime) || expirationTime.Sub(currentTime) <= 5*time.Minute {
			fmt.Println("Token is expired or expiring within 5 minutes.")
			// Delete the token from the map
			DeleteTokenInfo(spartanTokenPassedIn)
			// Get the refresh token from the map
			refreshtoken := tokenInfo.RefreshToken
			// Make the OAuth request
			spartanTokenResp, err := ProcessAuthCodeWithRefresh(refreshtoken, c.Writer, c.Request, OAuthExpiration)
			if err != nil {
				fmt.Println("Error when processing auth code with refresh token: ", err)
				fmt.Println("Signing user out")
				DeleteTokenInfo(spartanTokenPassedIn)
				return GamerInfo{XBLToken: "", SpartanKey: "", Gamertag: "", XUID: ""}
			}
			newGamerInfo, err := RequestUserProfile(spartanTokenResp.SpartanToken)
			if err != nil {
				fmt.Println("Error when getting refreshed user profile: ", err)
			}
			newGamerInfo.XBLToken = spartanTokenResp.XBLToken
			newGamerInfo.SpartanKey = spartanTokenResp.SpartanToken
			// remove old token info
			DeleteTokenInfo(spartanTokenPassedIn)
			return newGamerInfo

		} else {
			fmt.Println("Token is valid.")
			return gamerInfoPassedIn
		}
	}
	return gamerInfoPassedIn

}
