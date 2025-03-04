package consts

// The constants defined here generally consist of error codes and error descriptions, usually used for interface returns
const (
	// Process terminated
	ProcessKilled string = "Received signal, process terminated"
	// Form validator prefix
	ValidatorPrefix              string = "Form_Validator_"
	ValidatorParamsCheckFailCode int    = -400300
	ValidatorParamsCheckFailMsg  string = "Parameter validation failed"

	// Server code error
	ServerOccurredErrorCode int    = -500100
	ServerOccurredErrorMsg  string = "System error"
	GinSetTrustProxyError   string = "Gin set trust proxy server error"

	// Token related
	JwtTokenOK            int    = 200100                         // Token valid
	JwtTokenInvalid       int    = -400100                        // Invalid token
	JwtTokenExpired       int    = -400101                        // Expired token
	JwtTokenFormatErrCode int    = -400102                        // Submitted token format error
	JwtTokenFormatErrMsg  string = "Submitted token format error" // Submitted token format error
	JwtTokenMustValid     string = "Token must be valid, "        // Token must be valid

	// SnowFlake algorithm
	StartTimeStamp = int64(1483228800000) // Start timestamp (2017-01-01)
	MachineIdBits  = uint(10)             // Number of bits occupied by machine id
	SequenceBits   = uint(12)             // Number of bits occupied by sequence
	// MachineIdMax   = int64(-1 ^ (-1 << MachineIdBits)) // Maximum number of supported machine ids
	SequenceMask   = int64(-1 ^ (-1 << SequenceBits)) //
	MachineIdShift = SequenceBits                     // Machine id left shift bits
	TimestampShift = SequenceBits + MachineIdBits     // Timestamp left shift bits

	// CURD common business status codes
	CurdStatusOkCode         int    = 200
	CurdStatusOkMsg          string = "Success"
	CurdCreatFailCode        int    = -400200
	CurdCreatFailMsg         string = "Creation failed"
	CurdUpdateFailCode       int    = -400201
	CurdUpdateFailMsg        string = "Update failed"
	CurdDeleteFailCode       int    = -400202
	CurdDeleteFailMsg        string = "Deletion failed"
	CurdSelectFailCode       int    = -400203
	CurdSelectFailMsg        string = "No data found"
	CurdRegisterFailCode     int    = -400204
	CurdRegisterFailMsg      string = "Registration failed"
	CurdLoginFailCode        int    = -400205
	CurdLoginFailMsg         string = "Login failed"
	CurdRefreshTokenFailCode int    = -400206
	CurdRefreshTokenFailMsg  string = "Token refresh failed"

	// File upload
	FilesUploadFailCode            int    = -400250
	FilesUploadFailMsg             string = "File upload failed, error getting uploaded file!"
	FilesUploadMoreThanMaxSizeCode int    = -400251
	FilesUploadMoreThanMaxSizeMsg  string = "Uploaded file exceeds the maximum size allowed by the system, maximum size allowed by the system:"
	FilesUploadMimeTypeFailCode    int    = -400252
	FilesUploadMimeTypeFailMsg     string = "File mime type not allowed"

	// WebSocket
	WsServerNotStartCode int    = -400300
	WsServerNotStartMsg  string = "WebSocket service not started, please enable it in the configuration file, related path: config/config.yml"
	WsOpenFailCode       int    = -400301
	WsOpenFailMsg        string = "WebSocket open phase initialization of basic parameters failed"

	// Captcha
	CaptchaGetParamsInvalidMsg    string = "Get captcha: submitted captcha parameters are invalid, please check if the captcha ID and file name suffix are complete"
	CaptchaGetParamsInvalidCode   int    = -400350
	CaptchaCheckParamsInvalidMsg  string = "Check captcha: submitted parameters are invalid, please check if the key names of the submitted captcha ID and captcha value are consistent with the configuration items"
	CaptchaCheckParamsInvalidCode int    = -400351
	CaptchaCheckOkMsg             string = "Captcha check passed"
	CaptchaCheckFailCode          int    = -400355
	CaptchaCheckFailMsg           string = "Captcha check failed"
)

const NotNull = "not_null"

const (
	TraceIDKey = "_trace_id_"
	UserID     = "userId"
	UserIDKey  = "_user_id"
	TokenKey   = "token"
	UserInfo   = "userInfo"
)
