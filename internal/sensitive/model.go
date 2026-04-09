package sensitive

type SensitiveWord struct {
	ID   int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Word string `gorm:"column:word" json:"word"`
}

func (SensitiveWord) TableName() string {
	return "sensitive_words"
}
