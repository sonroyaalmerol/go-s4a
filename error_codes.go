package s4a

type ResultCode uint8

const (
	ResultSuccess          ResultCode = 0
	ResultScheduleError    ResultCode = 2
	ResultExceededLimit    ResultCode = 3
	ResultNoPermission     ResultCode = 4
	ResultReaderError      ResultCode = 5
	ResultExpired          ResultCode = 6
	ResultWorkModeDisabled ResultCode = 7
	ResultInternalError    ResultCode = 8
	ResultDecodeFailed     ResultCode = 9
	ResultGateTimeout      ResultCode = 10
	ResultAntiPassback     ResultCode = 11
	ResultNotSupported     ResultCode = 12
	ResultUnknownError     ResultCode = 13
	ResultFailed           ResultCode = 14
	ResultNotRegistered    ResultCode = 16
	ResultPasswordError    ResultCode = 17
	ResultInvalidSyncType  ResultCode = 18
	ResultInvalidSyncFmt   ResultCode = 19
	ResultSyncDataLimit    ResultCode = 20
	ResultInvalidSyncSeq   ResultCode = 21
	ResultNetUnknown       ResultCode = 22
	ResultNetDisconnected  ResultCode = 23
	ResultNetRestored      ResultCode = 24
	ResultNetRebootDevice  ResultCode = 25
	ResultNetRebootChip    ResultCode = 26
	ResultAntiCollision    ResultCode = 27
	ResultManualLock       ResultCode = 28
	ResultInterlock        ResultCode = 29
	ResultCardRWFailed     ResultCode = 30
	ResultGroupIDError     ResultCode = 31
	ResultSystemStatus     ResultCode = 32
	ResultBlacklist        ResultCode = 33
	ResultStorageError     ResultCode = 34
	ResultNotAuthorized    ResultCode = 35
	ResultTooManyInside    ResultCode = 36
	ResultAgeRestriction   ResultCode = 37
	ResultIDExpired        ResultCode = 38
)

func (c ResultCode) String() string {
	return ControllerErrorCode(uint8(c))
}

func ControllerErrorCode(code uint8) string {
	switch code {
	case 0:
		return "Success"
	case 2:
		return "Schedule error"
	case 3:
		return "Exceeded limit"
	case 4:
		return "No permission"
	case 5:
		return "Reader error"
	case 6:
		return "Expired"
	case 7:
		return "Work mode disabled"
	case 8:
		return "Internal error"
	case 9:
		return "Number decode failed"
	case 10:
		return "Gate timeout"
	case 11:
		return "Anti-passback"
	case 12:
		return "Not supported"
	case 13:
		return "Unknown error"
	case 14:
		return "Failed"
	case 16:
		return "Not registered / expired"
	case 17:
		return "Password error"
	case 18:
		return "Invalid sync type"
	case 19:
		return "Invalid sync message format"
	case 20:
		return "Sync data limit"
	case 21:
		return "Invalid sync data count/sequence"
	case 22:
		return "Network state unknown"
	case 23:
		return "Network disconnected"
	case 24:
		return "Network restored"
	case 25:
		return "Network check — reboot device"
	case 26:
		return "Network check — reboot chip"
	case 27:
		return "Anti-collision"
	case 28:
		return "Manual lock"
	case 29:
		return "Multi-door interlock"
	case 30:
		return "Card read/write failed"
	case 31:
		return "Group ID error"
	case 32:
		return "System status detail"
	case 33:
		return "Blacklist"
	case 34:
		return "Storage error"
	case 35:
		return "Not authorized"
	case 36:
		return "Too many people inside"
	case 37:
		return "Age restriction"
	case 38:
		return "ID expired"
	default:
		return "Unknown"
	}
}
