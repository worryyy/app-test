package sensitive

type wordQuery struct {
	Word string `form:"word" binding:"required"`
}
