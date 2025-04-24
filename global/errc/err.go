package errc

const (
	// System part
	ErrorsSystemError               string = "System error, please contact the administrator"
	ErrorsContainerKeyAlreadyExists string = "The key has already been registered in the container"
	ErrorsPublicNotExists           string = "public directory does not exist"
	ErrorsConfigYamlNotExists       string = "config.yml configuration file does not exist"
	ErrorsConfigGormNotExists       string = "gorm_v2.yml configuration file does not exist"
	ErrorsStorageLogsNotExists      string = "storage/logs directory does not exist"
	ErrorsConfigInitFail            string = "Error initializing configuration file"
	ErrorsSoftLinkCreateFail        string = "Failed to create soft link automatically, please run the client as an administrator (development environment is goland, production environment check the command executor permissions), " +
		"last possibility: if you are a 360 user, please exit the 360 related software to ensure that the go language creates soft link function: os.Symlink() runs normally"
	ErrorsSoftLinkDeleteFail string = "Failed to delete soft link"

	ErrorsFuncEventAlreadyExists string = "Failed to register function event, key name has already been registered"
	ErrorsFuncEventNotRegister   string = "No function found corresponding to the key name"
	ErrorsFuncEventNotCall       string = "The registered function cannot be executed correctly"
	ErrorsBasePath               string = "Failed to initialize project root directory"
	ErrorsTokenBaseInfo          string = "The most basic format error of the token, please provide a valid token!"
	ErrorsNoAuthorization        string = "Token authentication failed, please re-acquire the token through the token authorization interface"
	ErrorsRefreshTokenFail       string = "The token does not meet the refresh conditions, please re-acquire the token through the login interface!"
	ErrorsParseTokenFail         string = "Failed to parse token"
	ErrorsDBInitFail             string = "%s Database driver, connection initialization failed"
	ErrorsCasbinNoAuthorization  string = "Casbin authentication failed, please check the casbin setting parameters in the background"
	ErrorsNotInitGlobalPointer   string = "%s db connection not initialized"
	// Database part
	ErrorsDbDriverNotExists        string = "Database driver type does not exist, currently supported database types: mysql, sqlserver, postgresql, the database type you submitted:"
	ErrorsDialectorDbInitFail      string = "gorm dialector initialization failed, dbType:"
	ErrorsGormDBCreateParamsNotPtr string = "The parameter of gorm Create function must be a pointer"
	ErrorsGormDBUpdateParamsNotPtr string = "The parameters of gorm's Update, Save functions must be pointers (to perfectly support all callback functions of gorm, please add & before the parameter)"
	// Redis part
	ErrorsRedisInitConnFail string = "Failed to initialize redis connection pool"
	ErrorsRedisAuthFail     string = "Redis Auth authentication failed, wrong password"
	ErrorsRedisGetConnFail  string = "Failed to get a connection from the redis connection pool, exceeded the maximum retry count"
	// Form parameter validator errors
	ErrorsValidatorNotExists      string = "Validator does not exist"
	ErrorsValidatorTransInitFail  string = "Error initializing validator translator"
	ErrorNotAllParamsIsBlank      string = "This interface does not allow all parameters to be empty, please submit the required parameters according to the interface requirements"
	ErrorsValidatorBindParamsFail string = "Failed to bind parameters to validator"

	// Token part
	ErrorsTokenInvalid          string = "Invalid token"
	ErrorsTokenNotActiveYet     string = "Token is not valid yet"
	ErrorsTokenMalFormed        string = "Token is malformed"
	ErrorsTokenPermissionDenied string = "Insufficient permissions"

	ErrorsServicePermissionDenied string = "Insufficient service permissions"

	// Snowflake
	ErrorsSnowflakeGetIdFail string = "Error occurred while getting snowflake unique ID"
	// WebSocket
	ErrorsWebsocketOnOpenFail                 string = "Error occurred during websocket onopen phase"
	ErrorsWebsocketUpgradeFail                string = "Error occurred during websocket Upgrade protocol upgrade"
	ErrorsWebsocketReadMessageFail            string = "Error occurred in websocket ReadPump (real-time message reading) coroutine"
	ErrorsWebsocketBeatHeartFail              string = "Error occurred in websocket BeatHeart heartbeat coroutine"
	ErrorsWebsocketBeatHeartsMoreThanMaxTimes string = "Websocket BeatHeart failed more than the maximum number of times"
	ErrorsWebsocketSetWriteDeadlineFail       string = "Error setting websocket message write deadline"
	ErrorsWebsocketWriteMgsFail               string = "Websocket Write Msg (send msg) failed"
	ErrorsWebsocketStateInvalid               string = "Websocket state is no longer available (disconnected, stuck, etc., causing both parties to be unable to interact)"
	// RabbitMQ
	ErrorsRabbitMqReconnectFail string = "RabbitMQ consumer failed to reconnect after disconnection, exceeded the maximum number of attempts"

	// File upload
	ErrorsFilesUploadOpenFail string = "Failed to open file, details:"
	ErrorsFilesUploadReadFail string = "Failed to read 32 bytes of file, details:"

	// Casbin initialization possible errors
	ErrorCasbinCanNotUseDbPtr         string = "Casbin initialization is based on the gorm initialized database connection pointer, the program detected that the gorm connection pointer is invalid, please check the database configuration!"
	ErrorCasbinCreateAdaptFail        string = "Error occurred in casbin NewAdapterByDBUseTableName:"
	ErrorCasbinCreateEnforcerFail     string = "Error occurred in casbin NewEnforcer:"
	ErrorCasbinNewModelFromStringFail string = "Error occurred in NewModelFromString call:"
)
