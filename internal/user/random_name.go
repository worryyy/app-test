package user

import (
	"fmt"
	"math/rand"
	"time"
)

var humorousPrefix = []string{
	"奶茶", "懒觉", "快递", "表情包", "碎碎念",
	"游戏", "零食", "笑话", "拖延", "美梦",
	"彩虹屁", "拍照", "八卦", "发呆", "逛街", "momo",
}

var humorousSuffix = []string{
	"收割机", "守护者", "探险家", "大户", "专员",
	"躺赢侠", "大胃王", "制造机", "艺术家", "编织者",
	"专家", "做作精", "广播站", "专业户", "战斗机", "阿白",
}

var anonymousPrefix = []string{
	"蓬莱", "昆仑", "忘川", "长安", "广寒", "潇湘", "云梦", "归墟", "青丘", "北冥",
	"扶桑", "琅琊", "西洲", "幽州", "兰亭", "长亭", "断桥", "寒山", "紫禁", "云顶",
	"岛屿", "天台", "街角", "深海", "阁楼", "荒原", "极地", "花房", "书店", "长廊",
	"隧道", "梦境", "彼岸", "便利店", "美术馆", "海岸线", "无人区", "半岛", "银河", "极夜",
	"温室", "后海", "茶馆", "月球", "废墟", "钟楼", "灯塔", "迷宫", "雪原", "森林",
}

var anonymousSuffix = []string{
	"问道", "抚琴", "折扇", "听雨", "煮酒", "仗剑", "飞升", "入魔", "渡劫", "观星",
	"画眉", "题诗", "醉卧", "寻梅", "拜月", "御剑", "归隐", "焚香", "弄影", "听风",
	"发呆", "私奔", "邂逅", "沉溺", "逃离", "虚度", "放空", "治愈", "流浪", "定格",
	"独白", "微醺", "失联", "路过", "拾荒", "做梦", "热恋", "兜风", "漫游", "潜水",
	"失眠", "呼吸", "坠落", "拥抱", "告别", "回信", "看海", "散步", "幻想", "涂鸦",
}

func randomHumorousID() string {
	return pickRandom(humorousPrefix) + pickRandom(humorousSuffix)
}

func randomAnonymousID() string {
	return fmt.Sprintf("%s%sのe", pickRandom(anonymousPrefix), pickRandom(anonymousSuffix))
}

func pickRandom(items []string) string {
	if len(items) == 0 {
		return ""
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return items[r.Intn(len(items))]
}
