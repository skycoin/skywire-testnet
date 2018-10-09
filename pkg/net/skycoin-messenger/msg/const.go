package msg

const (
	MSG_OP_SIZE  = 1
	MSG_SEQ_SIZE = 4

	MSG_HEADER_BEGIN = 0
	MSG_OP_BEGIN
	MSG_OP_END = MSG_OP_BEGIN + MSG_OP_SIZE
	MSG_SEQ_BEGIN
	MSG_SEQ_END = MSG_SEQ_BEGIN + MSG_SEQ_SIZE

	MSG_HEADER_END
)

const (
	OP_ACCOUNT = iota // query created keys
	OP_REG // create key
	OP_LOGIN // use key to login
	OP_SEND // send msg to others
	OP_ACK // ack msg
	OP_SIZE
)
