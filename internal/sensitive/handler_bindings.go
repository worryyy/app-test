package sensitive

type wordQuery struct {
	Word string `form:"word" binding:"required"`
}

type updateWordQuery struct {
	Word       string `form:"word" binding:"required"`
	UpdateWord string `form:"updateWord" binding:"required"`
}

type pageQuery struct {
	Page int `form:"page" binding:"required,min=1"`
	Size int `form:"size" binding:"required,min=1"`
}
