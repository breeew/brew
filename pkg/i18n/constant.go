package i18n

var ALLOW_LANG = map[string]bool{
	"en":    true,
	"zh-CN": true,
}

const DEFAULT_LANG = "en"

const (
	ERROR_INTERNAL                   = "error.internal"
	ERROR_NOT_FOUND                  = "error.notfound"
	ERROR_INVALIDARGUMENT            = "error.invalidargument"
	ERROR_PERMISSION_DENIED          = "error.permission.denied"
	ERROR_UNAUTHORIZED               = "error.unauthorized"
	ERROR_EXIST                      = "error.exist"
	ERROR_TITLE_EXIST                = "error.title.exist"
	ERROR_FORBIDDEN                  = "error.forbidden"
	ERROR_TOO_MANY_REQUESTS          = "error.tooManyRequests"
	ERROR_UNSUPPORTED_FEATURE        = "error.unsupported.feature"
	ERROR_VERIFY_CODE_ALREADY_SENDED = "error.verifycodesended"
	ERROR_RESET_EMAIL_ALREADY_SENDED = "error.reset_email_sended"
	ERROR_VERIFY_CODE_INCORRECT      = "error.incorrect.verifycode"
	ERROR_LOGIN_ACCOUNT_INCORRECT    = "error.login.account.incorrect"
	ERROR_EMAIL_ALREADY_REGISTED     = "error.email_has_already_registed"
	ERROR_EMAIL_NOT_MATCH            = "error.email_not_match"
	ERROR_EMAIL_NOT_REGISTERED       = "error.email_not_registered"
	ERROR_ALREADY_INVITED            = "error.already_invited"

	ERROR_INVALID_TOKEN   = "error.invalid.token"
	ERROR_INVALID_ACCOUNT = "error.invalid.account"

	ERROR_LOGIC_VECTOR_DB_NOT_MATCHED_CONTENT_DB = "error.logic.vector.db.notmatch.content.db"
)
