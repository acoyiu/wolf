package wordlib

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
)

var words = map[Difficulty][]string{
	Easy: {
		"蘋果", "香蕉", "西瓜", "草莓", "葡萄", "橘子", "檸檬", "芒果", "鳳梨", "櫻桃",
		"太陽", "月亮", "星星", "雲朵", "彩虹", "閃電", "大海", "河流", "山脈", "森林",
		"書包", "鉛筆", "橡皮", "尺子", "桌子", "椅子", "黑板", "粉筆", "書本", "筆記",
		"貓咪", "小狗", "兔子", "金魚", "鸚鵡", "烏龜", "蝴蝶", "螞蟻", "蜜蜂", "蜻蜓",
		"汽車", "火車", "飛機", "輪船", "腳踏車", "公車", "地鐵", "摩托車", "計程車", "救護車",
		"蛋糕", "餅乾", "巧克力", "冰淇淋", "糖果", "果汁", "牛奶", "麵包", "三明治", "漢堡",
		"電視", "電話", "電腦", "冰箱", "洗衣機", "微波爐", "吹風機", "吸塵器", "電風扇", "冷氣",
		"足球", "籃球", "棒球", "網球", "羽毛球", "乒乓球", "游泳", "跑步", "跳繩", "溜冰",
		"醫生", "老師", "警察", "消防員", "廚師", "司機", "農夫", "漁夫", "護士", "郵差",
		"春天", "夏天", "秋天", "冬天", "聖誕節", "新年", "生日", "禮物", "氣球", "煙火",
	},
	Medium: {
		"民主", "引力", "咖啡因", "疫苗", "基因", "演算法", "生態系", "碳排放", "人工智慧", "機器學習",
		"通貨膨脹", "匯率", "股票", "債券", "供應鏈", "全球化", "關稅", "壟斷", "經濟衰退", "國內生產毛額",
		"光合作用", "細胞分裂", "化學反應", "牛頓定律", "電磁波", "原子核", "半導體", "超導體", "暗物質", "黑洞",
		"文藝復興", "工業革命", "冷戰", "絲綢之路", "大航海", "啟蒙運動", "法國大革命", "十字軍", "明治維新", "二戰",
		"社群媒體", "雲端運算", "大數據", "物聯網", "虛擬實境", "擴增實境", "無人機", "三維列印", "電子商務", "串流平台",
		"交響曲", "油畫", "雕塑", "芭蕾舞", "歌劇", "爵士樂", "搖滾樂", "嘻哈", "街舞", "即興表演",
		"哲學", "心理學", "社會學", "人類學", "考古學", "語言學", "倫理學", "邏輯學", "美學", "形上學",
		"光纖", "衛星", "雷達", "聲納", "望遠鏡", "顯微鏡", "加速器", "核反應爐", "太陽能板", "風力發電",
		"永續發展", "碳中和", "循環經濟", "綠色能源", "生態足跡", "環境影響", "有機農業", "水資源", "森林砍伐", "海洋污染",
		"談判", "外交", "協議", "制裁", "公投", "議會", "憲法", "司法", "人權", "主權",
	},
	Hard: {
		"量子纏結", "存在主義", "區塊鏈", "薛丁格的貓", "奧坎剃刀", "蝴蝶效應", "費曼圖", "哥德爾不完備", "圖靈測試", "納什均衡",
		"認知失調", "斯德哥爾摩症候群", "鄧寧克魯格", "馬斯洛需求", "巴甫洛夫", "佛洛伊德", "榮格原型", "從眾效應", "確認偏誤", "倖存者偏差",
		"黎曼猜想", "費馬最後", "龐加萊猜想", "哥德巴赫", "P與NP", "混沌理論", "碎形幾何", "拓撲學", "傅立葉轉換", "貝氏定理",
		"暗能量", "反物質", "弦理論", "多重宇宙", "蟲洞", "事件視界", "霍金輻射", "宇宙膨脹", "大霹靂", "中子星",
		"後現代主義", "解構主義", "虛無主義", "功利主義", "社會契約", "無政府主義", "資本論", "自由意志", "決定論", "相對主義",
		"基因編輯", "表觀遺傳", "蛋白質摺疊", "幹細胞", "微生物組", "免疫療法", "基因驅動", "合成生物", "端粒酶", "朊病毒",
		"零知識證明", "共識機制", "智能合約", "去中心化", "默克爾樹", "拜占庭容錯", "工作量證明", "權益證明", "側鏈", "分片技術",
		"深度學習", "生成對抗", "強化學習", "遷移學習", "注意力機制", "變分推論", "梯度消失", "過度擬合", "特徵工程", "超參數",
		"量子退相干", "量子霸權", "量子退火", "量子糾錯", "量子閘", "疊加態", "測不準原理", "波粒二象", "量子穿隧", "量子密鑰",
		"平行宇宙", "時間悖論", "費米悖論", "德雷克方程", "戴森球", "卡爾達肖夫", "奧爾特雲", "柯伊伯帶", "潮汐鎖定", "拉格朗日點",
	},
}

// Library holds the word library.
type Library struct {
	words map[Difficulty][]string
}

// New creates a new word Library with default words.
func New() *Library {
	return &Library{words: words}
}

// GetCandidates returns n random candidate words for the given difficulty.
func (l *Library) GetCandidates(difficulty Difficulty, n int) ([]string, error) {
	pool, ok := l.words[difficulty]
	if !ok {
		return nil, fmt.Errorf("invalid difficulty level")
	}
	if n > len(pool) {
		return nil, fmt.Errorf("not enough words in pool")
	}

	indices := make([]int, len(pool))
	for i := range indices {
		indices[i] = i
	}
	result := make([]string, n)
	for i := 0; i < n; i++ {
		max := big.NewInt(int64(len(indices) - i))
		j, err := rand.Int(rand.Reader, max)
		if err != nil {
			return nil, fmt.Errorf("random generation failed: %w", err)
		}
		idx := int(j.Int64()) + i
		indices[i], indices[idx] = indices[idx], indices[i]
		result[i] = pool[indices[i]]
	}
	return result, nil
}

// WordCount returns the number of words for a given difficulty.
func (l *Library) WordCount(difficulty Difficulty) (int, error) {
	pool, ok := l.words[difficulty]
	if !ok {
		return 0, fmt.Errorf("invalid difficulty level")
	}
	return len(pool), nil
}

// ValidDifficulties returns all valid difficulty levels.
func ValidDifficulties() []Difficulty {
	return []Difficulty{Easy, Medium, Hard}
}
