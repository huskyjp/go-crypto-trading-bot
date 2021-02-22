package models

import (
	"go-trading-bot/config"
	"sort"
)

type TradeParams struct {
	EmaEnable        bool
	EmaPeriod1       int
	EmaPeriod2       int
	BbEnable         bool
	BbN              int
	BbK              float64
	IchimokuEnable   bool
	MacdEnable       bool
	MacdFastPeriod   int
	MacdSlowPeriod   int
	MacdSignalPeriod int
	RsiEnable        bool
	RsiPeriod        int
	RsiBuyThread     float64
	RsiSellThread    float64
}

type Ranking struct {
	Enable      bool
	Performance float64
}

func (df *DataFrameCandle) OptimizeParams() *TradeParams {
	emaPerformance, emaPeriod1, emaPeriod2 := df.OptimizeEma()
	bbPerformance, bbN, bbK := df.OptimizeBb()
	macdPerformance, macdFastPeriod, macdSlowPeriod, macdSignalPeriod := df.OptimizeMacd()
	ichimokuPerforamcne := df.OptimizeIchimoku()
	rsiPerformance, rsiPeriod, rsiBuyThread, rsiSellThread := df.OptimizeRsi()

	emaRanking := &Ranking{false, emaPerformance}
	bbRanking := &Ranking{false, bbPerformance}
	macdRanking := &Ranking{false, macdPerformance}
	ichimokuRanking := &Ranking{false, ichimokuPerforamcne}
	rsiRanking := &Ranking{false, rsiPerformance}

	rankings := []*Ranking{emaRanking, bbRanking, macdRanking, ichimokuRanking, rsiRanking}
	sort.Slice(rankings, func(i, j int) bool { return rankings[i].Performance > rankings[j].Performance })

	isEnable := false
	for i, ranking := range rankings {
		if i >= config.Config.NumRanking {
			break
		}
		if ranking.Performance > 0 {
			ranking.Enable = true
			isEnable = true
		}
	}
	if !isEnable {
		return nil
	}

	tradeParams := &TradeParams{
		EmaEnable:        emaRanking.Enable,
		EmaPeriod1:       emaPeriod1,
		EmaPeriod2:       emaPeriod2,
		BbEnable:         bbRanking.Enable,
		BbN:              bbN,
		BbK:              bbK,
		IchimokuEnable:   ichimokuRanking.Enable,
		MacdEnable:       macdRanking.Enable,
		MacdFastPeriod:   macdFastPeriod,
		MacdSlowPeriod:   macdSlowPeriod,
		MacdSignalPeriod: macdSignalPeriod,
		RsiEnable:        rsiRanking.Enable,
		RsiPeriod:        rsiPeriod,
		RsiBuyThread:     rsiBuyThread,
		RsiSellThread:    rsiSellThread,
	}
	return tradeParams
}
