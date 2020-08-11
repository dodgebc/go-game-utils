package sgfgrab

import (
	"testing"
)

func BenchmarkAlphaGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Grab(alphaGoSgfText)
	}
}

func BenchmarkOgs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Grab(ogsSgfText)
	}
}

func TestAlphaGo(t *testing.T) {
	gs, err := Grab(alphaGoSgfText)
	if err != nil {
		t.Error(err)
	}
	if len(gs) != 1 {
		t.Fatalf("got %d games, want 1", len(gs))
	}
	if !gs[0].Equals(alphaGoGameData) {
		t.Errorf("\ngot:\n%#v\n\nwant:\n%#v", gs[0], alphaGoGameData)
	}
}

func TestOgs(t *testing.T) {
	gs, err := Grab(ogsSgfText)
	if err != nil {
		t.Error(err)
	}
	if len(gs) != 1 {
		t.Fatalf("got %d games, want 1", len(gs))
	}
	if !gs[0].Equals(ogsGameData) {
		t.Errorf("\ngot:\n%#v\n\nwant:\n%#v", gs[0], ogsGameData)
	}
}

func TestParseError(t *testing.T) {
	sgfText := "(SZ[3:2:1])"
	if _, err := Grab(sgfText); err == nil {
		t.Error("no error on bad size")
	}

	sgfText = "(KM[12.5a])"
	if _, err := Grab(sgfText); err == nil {
		t.Error("no error on bad komi")
	}

	sgfText = "(KM[nan])"
	if _, err := Grab(sgfText); err == nil {
		t.Error("no error on nan komi")
	}

	sgfText = "(HA[-4])"
	if _, err := Grab(sgfText); err == nil {
		t.Error("no error on bad handicap")
	}

	sgfText = "(RE[Z+10.5])"
	if g, err := Grab(sgfText); err != nil || len(g) != 1 || g[0].Winner != "" || g[0].Score != 0.0 || g[0].End != "" {
		t.Error("bad result not ignored")
	}

	sgfText = "(BR[341p])"
	if g, _ := Grab(sgfText); len(g) != 1 || g[0].BlackRank != "" {
		t.Error("bad rank not ignored")
	}

	sgfText = "(TM[20.5])"
	if g, _ := Grab(sgfText); len(g) != 1 || g[0].Time != 0 {
		t.Error("bad time not ignored")
	}

	sgfText = "(DT[202])"
	if g, _ := Grab(sgfText); len(g) != 1 || g[0].Year != 0 {
		t.Error("bad date not ignored")
	}

	sgfText = "(B[Zc2])"
	if _, err := Grab(sgfText); err == nil {
		t.Error("no error on bad move")
	}
}

func TestHandicapCheck(t *testing.T) {
	sgfText := "(;HA[1];AB[aa];AB[ab])"
	_, err := Grab(sgfText)
	if err == nil {
		t.Error("no error with too many setup stones")
	}

	sgfText = "(;HA[1];AW[aa])"
	_, err = Grab(sgfText)
	if err == nil {
		t.Error("no error with white setup stone")
	}

	sgfText = "(;AB[bb])"
	g, err := Grab(sgfText)
	if (err != nil) || (len(g) != 1) || (g[0].Handicap != 1) {
		t.Error("no handicap not corrected")
	}

	sgfText = "(;HA[1]B[ab])"
	g, err = Grab(sgfText)
	if (err != nil) || (len(g) != 1) || (g[0].Handicap != 1) || (len(g[0].Setup) != 1) || (g[0].Setup[0] != "Bab") {
		t.Error("mislabeled setup stones not corrected")
	}
}

func TestRootProperties(t *testing.T) {
	sgfText := "(;SZ[9:10]KM[0.5]HA[2]RE[B+20.5]PB[me]PW[you]BR[4k]WR[9p]TM[200]OT[something]DT[2020-01-01];AB[cc][dd](;B[ab];W[bA];B[tt](;W[])(;W[cc])))"
	gs, err := Grab(sgfText)
	if err != nil {
		t.Error(err)
	}
	if len(gs) != 1 {
		t.Fatalf("got %d games, want 1", len(gs))
	}
	g := gs[0]
	expect := GameData{
		Size:        [2]int{10, 9},
		Komi:        0.5,
		Handicap:    2,
		Winner:      "B",
		Score:       20.5,
		End:         "Scored",
		BlackPlayer: "me",
		WhitePlayer: "you",
		BlackRank:   "4k",
		WhiteRank:   "9p",
		Time:        200,
		Year:        2020,
		Moves:       []string{"Bab", "WbA", "B", "W"},
		Setup:       []string{"Bcc", "Bdd"},
	}
	if !g.Equals(expect) {
		t.Errorf("\ngot:\n%#v\n\nwant:\n%#v", g, expect)
	}
}

func TestMultipleGames(t *testing.T) {
	sgfText := "(;SZ[3:2])(;SZ[9])"
	gs, err := Grab(sgfText)
	if err != nil {
		t.Error(err)
	}
	if len(gs) != 2 {
		t.Fatalf("got %d games, want 2", len(gs))
	}
	expect := GameData{Size: [2]int{2, 3}}
	if !gs[0].Equals(expect) {
		t.Errorf("\ngot:\n%#v\n\nwant:\n%#v", gs[0], expect)
	}
	expect = GameData{Size: [2]int{9, 9}}
	if !gs[1].Equals(expect) {
		t.Errorf("\ngot:\n%#v\n\nwant:\n%#v", gs[1], expect)
	}
}

func TestTree(t *testing.T) {
	sgfText := "(;GM[1]FF[4]CA[UTF-8](;B[ab](;W[cd];B[ce]);W[ae])(;B[ba]))"
	gs, err := Grab(sgfText)
	if err != nil {
		t.Error(err)
	}
	if len(gs) != 1 {
		t.Fatalf("got %d games, want 1", len(gs))
	}
	g := gs[0]
	expect := GameData{
		Size:  [2]int{19, 19},
		Moves: []string{"Bab", "Wcd", "Bce"},
	}
	if !g.Equals(expect) {
		t.Errorf("\ngot:\n%#v\n\nwant:\n%#v", g, expect)
	}
}

func TestNoGame(t *testing.T) {
	sgfText := ";SZ[19]"
	gs, err := Grab(sgfText)
	if err != nil {
		t.Error(err)
	}
	if len(gs) != 0 {
		t.Fatalf("got %d games, want 1", len(gs))
	}
}

func TestTrivial(t *testing.T) {
	sgfText := "(;GM[1]FF[4]CA[UTF-8])"
	gs, err := Grab(sgfText)
	if err != nil {
		t.Error(err)
	}
	if len(gs) != 1 {
		t.Fatalf("got %d games, want 1", len(gs))
	}
	g := gs[0]
	expect := GameData{Size: [2]int{19, 19}}
	if !g.Equals(expect) {
		t.Errorf("\ngot:\n%#v\n\nwant:\n%#v", g, expect)
	}
}

var alphaGoSgfText string = `(;GM[1]FF[4]CA[UTF-8]AP[CGoban:3]ST[2]
	RU[Chinese]SZ[19]KM[7.50]TM[7200]OT[3x60 byo-yomi]
	PW[Lee Sedol]PB[AlphaGo]WR[9p]DT[2016-03-13]C[Game 4 - Endurance
	
	Commentary by Fan Hui 2p
	Expert Go analysis by Gu Li 9p and Zhou Ruiyang 9p
	Translated by Lucas Baker, Thomas Hubert, and Thore Graepel
	
	When I arrived in the playing room on the morning of the fourth game, everyone appeared more relaxed than before. Whatever happened today, the match had already been decided. Nonetheless, there were still two games to play, and a true professional like Lee must give his all each time he sits down at the board.
	
	When Lee arrived in the playing room, he looked serene, the burden of expectation lifted. Perhaps he would finally recover his composure, forget his surroundings, and simply play his best Go. One thing was certain: it was not in Lee's nature to surrender.
	
	There were many fewer reporters in the press room this time. It seemed the media thought the interesting part was finished, and the match was headed for a final score of 5-0. But in a game of Go, although the information is open for all to see, often the results defy our expectations. Until the final stone is played, anything is possible.]RE[W+Resign]
	;B[pd]
	;W[dp]
	;B[cd]
	;W[qp]
	;B[op]
	;W[oq]
	;B[nq]
	;W[pq]
	;B[cn]
	;W[fq]
	;B[mp]C[For the fourth game, Lee took White. Up to move 11, the opening was the same as the second game. AlphaGo is an extremely consistent player, and once it thinks a move is good, its opinion will not change.
	]
	;W[po]LB[ed:B][qn:A]C[During the commentary for game 2, I mentioned that AlphaGo prefers to tenuki with 12 and approach at B. Previously, Lee chose to finish the joseki normally at A. White 12, however, is an interesting alternative! Perhaps it was a test: would AlphaGo still tenuki as in game 2?]
	;B[iq]C[This time, AlphaGo chose the ordinary extension. After this move, AlphaGo's win rate was 50.5%. Both players had 1 hour and 52 minutes apiece.]
	;W[ec]C[White played the usual corner approach.]
	;B[hd]
	(;W[cg]C[Against Black's pincer, Lee counter-pincered to emphasize the left side. However, AlphaGo preferred the press as shown in the variation.]
	;B[ed]
	;W[cj]
	;B[dc]
	;W[bp]
	;B[nc]C[At this point, Lee had 1 hour and 40 minutes left, AlphaGo 1 hour and 47 minutes.]
	;W[qi]C[The opening had been a balanced one, and AlphaGo's win rate stood at 53%.]
	;B[ep]C[The better people got to know AlphaGo, the more started calling it by friendly nicknames, such as "Master A" in China and "Master Al" in Korea. Although such names mean little on their own, they represented a growing acceptance of AlphaGo as a partner and teacher, together with whom we could advance our understanding.
	
	During this competition, "Master A" never ceased to surprise, playing at least one extraordinary new move in every game. This move was the first such display in game 4.
	
	Theoretically, Black 23 is quite vulgar, as it induces White to strengthen the corner. However, since White's lower side is already solid, if this move helps Black on the outside, is it really as crude as it looks? See the variation.
	
	One of the major ideas in Chinese philosophy, from kung fu to Taoism, is the notion that "formlessness defeats form." This is not to suggest that one should do nothing, but rather that one should be ready to make use of any resource at any moment. In other words, complete flexibility is the surest path to success. When one is not committed to any style, there are no weak points at which to aim. Naturally, reaching this state demands a strong foundation and a formidable repertoire. AlphaGo seemed to have attained this "formless" style already: simple, easy to understand, and totally unexploitable.]
	(;W[eo]
	;B[dk]C[Lee chose the outside hane, and just as I was reading out the cut, AlphaGo changed course with the shoulder hit at 25. Lee cracked a smile, as if he were looking at a naughty child. AlphaGo was being mischievous indeed!]
	(;W[fp]C[Coolly, Lee protected the corner with 26. Most players found this move too slow, and even AlphaGo thought White should respond with the crawl as shown in the variation. Nonetheless, I saw this move as a sign that Lee had finally found the confidence to play his own game, regardless of anyone's approval. This was the Lee Sedol I knew: the wolf that, starving in the winter winds, still waited for his prey to come closer, biding his time for the moment his intuition knew would come.]
	;B[ck]C[When Black blocked here, AlphaGo’s win rate was 55%.]
	;W[dj]
	;B[ej]
	;W[ei]
	;B[fi]C[Perhaps because it was playing Black, AlphaGo played very aggressively during this game. With the double hane at 29 and 31, it seemed as if Black was attempting to completely overwhelm White.]
	;W[eh]
	;B[fh]
	(;W[bj]C[Even more shocking is that Lee submitted to it! Meek as a lamb, he let AlphaGo blockade the center and seal in White's group on the left side.
	
	Sometimes to endure is difficult, sometimes it is irrational, and sometimes it is futile. Whatever it meant, Lee endured, and perhaps this was way of showing us his unwavering faith and unwillingness to surrender.
	
	However, White did not need to suffer quite so badly. See the variation.]
	;B[fk]
	;W[fg]
	;B[gg]
	;W[ff]
	;B[gf]C[At this point, AlphaGo's win rate reached 60%.]
	;W[mc]C[After Black had constructed such a thick wall, it became urgent for White to invade the top. The attachment at 40 was a classic technique to do just that.]
	;B[md]C[At this point, Lee Sedol had 1 hour and 15 minutes left, AlphaGo 1 hour and 35 minutes.]
	;W[lc]
	;B[nb]
	(;W[id]C[AlphaGo thought this move was problematic. In view of Black's thickness, White's most attractive option would have been to live on the spot, as shown in the variation.]
	;B[hc]
	;W[jg]
	;B[pj]C[Compared to the previous games, Lee appeared much more relaxed and focused. Gone were the sighs, and the shaking of the head. Instead, he wore a look of intense concentration, as if waiting for something to arrive.
	
	White gracefully leapt out into the center, but Black initiated a magnificent leaning attack!]
	;W[pi]
	;B[oj]
	;W[oi]
	;B[ni]C[The greater the pressure became on White's top group, the more time Lee took to ponder each move. When Black played 51, Lee hesitated even longer. If Black dared to play so boldly, choosing the hane with only two stones to White's three, I could not imagine any move for White except the cut! Even AlphaGo thought cutting was the only move.]
	(;W[nh]C[But Lee continued to endure! Against Black's hane, he haned in return.]
	;B[mh]
	;W[ng]C[Against the double hane, he extended!
	
	"I can feel Lee's conviction," I wrote in my notebook, "waiting for the critical moment. But will that moment ever come?" As the game progressed, it seemed to everyone that Lee was once again on the verge of defeat.
	
	At this point, Lee's clock had 51 minutes, AlphaGo's 1 hour and 28 minutes.]
	;B[mg]
	;W[mi]
	;B[nj]
	;W[mf]
	;B[li]C[When Black ataried at 59, White had the option of linking up the group on the top. See the variation.]
	(;W[ne]
	;B[nd]
	;W[mj]C[Instead, however, Lee chose to pull out his center stone. With White's group now isolated and in grave peril, Lee's heart must have been overwhelmed with emotions. But perhaps he also sensed the long-awaited moment at hand.]
	;B[lf]C[Lee now had 42 minutes, AlphaGo 1 hour and 22 minutes.]
	;W[mk]
	;B[me]
	;W[nf]
	;B[lh]
	;W[qj]C[At this point, people suddenly began to congregate near the playing room. The rumor was spreading that the game was about to end, with AlphaGo victorious as expected. But Lee looked cool-headed as ever as he played the turn at 68. Was he really not afraid of dying?]
	;B[kk]C[AlphaGo continued striding forward with the knight's move, aiming to swallow the white group whole. Lee pondered deeply over his next move. If White was to have any chance, he would have to seize it now.
	
	At this point, Lee had 34 minutes, AlphaGo 1 hour and 19 minutes.]
	;W[ik]
	;B[ji]C[When Black closed off the center with 71, Lee had 27 minutes, AlphaGo 1 hour and 17 minutes.]
	;W[gh]
	;B[hj]C[After ten minutes of thought, Lee played the cut. Right on schedule, one minute later, AlphaGo enclosed White's cutting stone. Lee sighed, and kept thinking.]
	;W[ge]
	;B[he]
	;W[fd]
	;B[fc]C[At move 77, Lee had only 11 minutes left. Moreover, AlphaGo’s win rate had climbed over 70%. It seemed the game was over. 
	
	In fact, Lee had just completed the last of the preparations for his final charge!]
	;W[ki]C[At last, Lee Sedol launched his attack. Like an earthquake, the wedge at 78 tore apart the cracks in Black's fortress! None of us had anticipated this. When Gu Li saw White 78 from his broadcasting studio in China, he shouted: "The divine move!" All of Lee's painstaking preparations were finally about to bear fruit.
	
	Actually, Lee spent very little time on this move itself. Later, during the press conference, he told the assembled reporters that he had not spent much time calculating. He had simply played what felt right.
	
	Without a doubt, this move is a spectacular flash of insight - but does it really work? See the variation for an explanation of White's plan and Black's best response.
	
	Regardless, this move cast AlphaGo into complete confusion.]
	(;B[jj]
	;W[lj]C[Move 78 might not really work, but AlphaGo was at a complete loss to deal with it. When Black pulled back, White blocked at 80, and Black could no longer kill the white stones unconditionally.]
	;B[kh]
	;W[jh]C[Everybody still thought AlphaGo was aiming for the ko shown in the variation, but this was not the case.]
	(;B[ml]
	;W[nk]
	;B[ol]C[Inexplicably, AlphaGo began trying to extract the dead stones on the right side!]
	;W[ok]
	;B[pk]
	;W[pl]
	;B[qk]
	;W[nl]
	;B[kj]
	;W[ii]C[When White haned here, it was already difficult for Black to contain the white stones.
	
	At this point, AlphaGo’s win rate was in free fall. When White played 92, it dropped all the way to 55%, a full 15 points lower than before! What was going on?!]
	;B[rk]
	;W[om]LB[rk:A][om:B]C[AlphaGo seemed to have gone crazy, and began thrashing around wildly. The exchange of A for B reinforced White's center with no compensation.]
	;B[pg]
	;W[ql]
	;B[cp]LB[ki:A]C[This wedge was completely beyond understanding! Aja Huang, who had been the picture of calmness since the beginning of the match, now looked at me as if to ask, "What's happening!?" I answered with a look that said, "I don't know."
	
	Even now, we still do not know why AlphaGo lost its mind, playing senseless blunders one after another. Only one thing is certain: the original cause was the wedge at A, Lee Sedol's mystical "divine move." White 78 was incontrovertible proof of his determination to fight on, and his perseverance was rewarded with victory.]
	;W[co]
	;B[oe]
	;W[rl]
	;B[sk]
	;W[rj]
	;B[hg]C[Little by little, AlphaGo recovered its sanity, but too late to save the game. When Black played 103, AlphaGo’s win rate had fallen to 30%, the first time such low numbers had appeared since the match began.
	
	At this point, Lee entered byo-yomi, but for the first time, he was seeing the light of victory.
	
	The press room had been half-empty for some time, but now people began to pour in. Many reporters had already left, but when they heard that Lee Sedol might win, they turned around and came rushing back!]
	;W[ij]
	;B[km]
	;W[gi]
	;B[fj]
	;W[jl]
	;B[kl]
	;W[gl]
	;B[fl]
	;W[gm]
	;B[ch]
	;W[ee]
	;B[eb]
	;W[bg]
	;B[dg]
	;W[eg]
	;B[en]
	;W[fo]
	;B[df]
	;W[dh]
	;B[im]
	;W[hk]
	;B[bn]
	;W[if]
	;B[gd]
	;W[fe]
	;B[hf]
	;W[ih]
	;B[bh]
	;W[ci]
	;B[ho]
	;W[go]
	;B[or]
	;W[rg]C[Everyone was anxiously awaiting the news of Lee's victory. As the professional commentators grew more and more certain of the conclusion, the game continued to progress onscreen. Playing in byo-yomi, Lee continued to answer AlphaGo's every move with utmost caution.]
	;B[dn]
	;W[cq]
	;B[pr]
	;W[qr]
	;B[rf]
	;W[qg]
	;B[qf]
	;W[jc]
	;B[gr]
	;W[sf]
	;B[se]
	;W[sg]
	;B[rd]
	;W[bl]
	;B[bk]
	;W[ak]
	;B[cl]
	;W[hn]
	;B[in]
	;W[hp]
	;B[fr]
	;W[er]
	;B[es]
	;W[ds]
	;B[ah]
	;W[ai]
	;B[kd]
	;W[ie]
	;B[kc]
	;W[kb]
	;B[gk]
	;W[ib]
	;B[qh]
	;W[rh]
	;B[qs]
	;W[rs]
	;B[oh]
	;W[sl]
	;B[of]
	;W[sj]
	;B[ni]
	;W[nj]
	;B[oo]
	;W[jp]C[At move 180, AlphaGo finally resigned.
	
	Lee Sedol entered the press room to thunderous applause. At long last, he could smile. It was a simple, modest smile, nothing world-changing - but then, aren't little moments like this the real source of joy in life?])
	(;B[hh]C[Black should simply protect here.]
	;W[lg]
	;B[kg]
	;W[kf]
	;B[kj]
	;W[le]
	;B[ki]
	;W[lg]
	;B[ol]C[White can use the throw-in to connect in ko, but Black can save the four stones on the right with 12, and the game remains difficult for White.]))
	(;B[kj]C[White’s plan is simple: If Black answers this way, White will use the aji of the surroundings to kill the three black stones.]
	;W[ii]
	;B[ij]
	;W[gi]
	;B[fj]
	;W[jh]
	(;B[kh]C[If Black captures here, White's plan works - but what if Black had defended instead?]
	;W[hf]
	;B[hg]
	;W[if]C[Up to here, the three stones are captured, White is connected, and Black is lost.])
	(;B[gj]LB[gj:A]C[We shortly discovered that move 78 does not actually work! When White ataris, Black need not capture, but can connect instead at A. Due to the shortage of liberties, the sequence in the previous diagram fails.]
	;W[jj]
	;B[jk]
	;W[hf]
	;B[hg]
	;W[if]
	;B[hi]C[White is obliterated.])))
	(;W[nf]C[AlphaGo thinks White's best strategy is to connect up the top side.]
	;B[mj]C[Black would take the lone white stone.]
	;W[ld]
	;B[re]C[White saves everything, but Black retains a clear lead by defending the corner.]))
	(;W[fd]
	;B[fc]
	;W[nj]C[AlphaGo thinks White should cut.]
	;B[qj]
	;W[nk]
	;B[ri]
	;W[rh]
	;B[rj]
	;W[nh]
	;B[mi]
	;W[mh]
	;B[li]
	;W[lh]
	;B[ki]
	;W[qg]
	;B[pm]C[The result is still a chaotic fight in the center.]))
	(;W[ld]
	;B[me]
	;W[ib]C[AlphaGo thinks White should strive to make a base.]
	;B[gb]
	;W[id]
	;B[kf]
	;W[hc]
	;B[gd]
	;W[je]C[Now White is alive.]))
	(;W[ek]C[AlphaGo thinks White should fight back with the atari and hane.]
	;B[fj]
	;W[bk]
	;B[el]
	;W[lq]
	;B[lp]
	;W[nr]
	;B[kq]
	;W[mq]
	;B[jn]
	;W[fd]
	;B[ee]
	;W[jc]LB[jc:14][fd:12][ee:13][fh:1][fj:3][bk:4][ek:2][el:5][jn:11][lp:7][kq:9][lq:6][mq:10][nr:8]C[Black captures 2 in a ladder, but White can use the ladder breaker at 6 to invade the bottom. After 11, the bottom side is settled, and White can make use of the aji at the top to launch an invasion with 12 and 14. This way leads to a complicated battle.]))
	(;W[ck]TR[ep]C[AlphaGo thinks White should crawl with 2. Black will now set the marked stone into motion.]
	;B[fp]
	;W[eq]
	;B[do]
	;W[co]
	;B[dn]
	;W[gp]
	;B[fo]
	;W[hq]
	;B[ip]
	;W[go]
	;B[fn]
	;W[gn]
	;B[fm]
	;W[bn]C[Zhou Ruiyang showed a similar variation in his own livestream.]))
	(;W[eq]
	;B[do]
	;W[cp]
	;B[dk]LB[do:3][cp:4][ep:1][eq:2]C[It is difficult to judge the pros and cons of exchanging 1 through 4, and professional commentators ventured a wide range of opinions.]
	;W[ck]
	;B[cl]C[AlphaGo was planning to sacrifice a stone on the left to squeeze White.]
	;W[dl]
	;B[el]
	;W[dm]
	;B[em]
	;W[cm]
	;B[bm]
	;W[bl]
	;B[dn]
	;W[cl]
	;B[gp]
	;W[fp]
	;B[go]C[Finally, Black encloses the corner. This sequence looks interesting for Black.]))
	(;W[de]C[AlphaGo suggests pressing with the knight's move, leading to a fight on the left side.]
	;B[ce]
	;W[df]
	;B[cg]
	;W[dg]
	;B[ch]
	;W[dh]
	;B[di]
	;W[ei]
	;B[ej]
	;W[cf]
	;B[bf]
	;W[be]
	;B[bg]
	;W[cc]
	;B[bd]
	;W[dj]
	;B[ci]
	;W[fi]
	;B[fj]
	;W[gj]
	;B[gi]
	;W[fg]
	;B[hi]
	;W[gk]
	;B[fl]
	;W[dm]C[This by no means represents the best moves for both sides, but perhaps we can experiment with this strategy in future games.]))
	`

var ogsSgfText string = `(;FF[4]
	CA[UTF-8]
	GM[1]
	DT[2019-06-17]
	PC[OGS: https://online-go.com/game/18309555]
	GN[Friendly Match]
	PB[Spectral-7k]
	PW[Spectral-10k]
	BR[23k]
	WR[19k]
	TM[600]OT[5x30 byo-yomi]
	RE[B+2.5]
	SZ[19]
	KM[7.5]
	RU[Chinese]
	C[Spectral-7k: Hi! This bot is intended to play human-like games at any kyu rank, running on only a Raspberry Pi. Type "options" for more information. Good luck!
	Spectral-10k: Hi! This bot is intended to play human-like games at any kyu rank, running on only a Raspberry Pi. Type "options" for more information. Good luck!
	Spectral-7k: Thanks for the game! Always looking to improve, so feel free to message me with any problems, observations, or suggestions.
	Spectral-10k: Thanks for the game! Always looking to improve, so feel free to message me with any problems, observations, or suggestions.
	]
	;B[pd]
	(;W[dd]
	(;B[dp]
	(;W[pp]
	(;B[qn]
	(;W[nq]
	(;B[pj]
	(;W[nc]
	(;B[lc]
	(;W[qc]
	(;B[qd]
	(;W[pc]
	(;B[od]
	(;W[nb]
	(;B[me]
	(;W[cn]
	(;B[fq]
	(;W[dj]
	(;B[fc]
	(;W[cf]
	(;B[db]
	(;W[cc]
	(;B[hd]
	(;W[ql]
	(;B[pl]
	(;W[qk]
	(;B[pk]
	(;W[qj]
	(;B[qi]
	(;W[ri]
	(;B[qh]
	(;W[rh]
	(;B[qg]
	(;W[qm]
	(;B[pm]
	(;W[pn]
	(;B[po]
	(;W[on]
	(;B[qp]
	(;W[oo]
	(;B[qo]
	(;W[pq]
	(;B[qq]
	(;W[rn]
	(;B[ro]
	(;W[rm]
	(;B[pr]
	(;W[or]
	(;B[qr]
	(;W[hq]
	(;B[bp]
	(;W[ho]
	(;B[cl]
	(;W[en]
	(;B[el]
	(;W[fj]
	(;B[gl]
	(;W[hj]
	(;B[il]
	(;W[jj]
	(;B[kl]
	(;W[lj]
	(;B[ml]
	(;W[lm]
	(;B[ll]
	(;W[jm]
	(;B[jl]
	(;W[hm]
	(;B[hl]
	(;W[fm]
	(;B[fl]
	(;W[km]
	(;B[mm]
	(;W[mn]
	(;B[im]
	(;W[in]
	(;B[gm]
	(;W[gn]
	(;B[hn]
	(;W[io]
	(;B[go]
	(;W[hm]
	(;B[fo]
	(;W[fn]
	(;B[hn]
	(;W[eo]
	(;B[ep]
	(;W[hm]
	(;B[ln]
	(;W[lo]
	(;B[hn]
	(;W[gp]
	(;B[fp]
	(;W[hm]
	(;B[jn]
	(;W[kn]
	(;B[hn]
	(;W[co]
	(;B[hm]
	(;W[cp]
	(;B[cq]
	(;W[bo]
	(;B[bq]
	(;W[bl]
	(;B[bk]
	(;W[ck]
	(;B[dk]
	(;W[cj]
	(;B[bm]
	(;W[bj]
	(;B[al]
	(;W[ek]
	(;B[dl]
	(;W[oi]
	(;B[pi]
	(;W[nk]
	(;B[nl]
	(;W[jc]
	(;B[kd]
	(;W[lb]
	(;B[kb]
	(;W[kc]
	(;B[mb]
	(;W[ld]
	(;B[la]
	(;W[mc]
	(;B[lb]
	(;W[jb]
	(;B[md]
	(;W[le]
	(;B[jd]
	(;W[lf]
	(;B[ic]
	(;W[ib]
	(;B[hb]
	(;W[ja]
	(;B[ha]
	(;W[na]
	(;B[ia]
	(;W[mf]
	(;B[of]
	(;W[nf]
	(;B[oe]
	(;W[ie]
	(;B[he]
	(;W[if]
	(;B[hf]
	(;W[ig]
	(;B[hg]
	(;W[hh]
	(;B[gh]
	(;W[gi]
	(;B[fh]
	(;W[ff]
	(;B[ef]
	(;W[ee]
	(;B[eg]
	(;W[fd]
	(;B[ec]
	(;W[cb]
	(;B[cg]
	(;W[bg]
	(;B[ch]
	(;W[bh]
	(;B[df]
	(;W[de]
	(;B[ei]
	(;W[ej]
	(;B[ci]
	(;W[bi]
	(;B[di]
	(;W[gr]
	(;B[fr]
	(;W[gq]
	(;B[gs]
	(;W[hs]
	(;B[fs]
	(;W[ir]
	(;B[rg]
	(;W[rk]
	(;B[rc]
	(;W[rb]
	(;B[rd]
	(;W[sb]
	(;B[oc]
	(;W[ob]
	(;B[oh]
	(;W[nh]
	(;B[ni]
	(;W[mi]
	(;B[oj]
	(;W[nj]
	(;B[ng]
	(;W[mh]
	(;B[ca]
	(;W[ba]
	(;B[da]
	(;W[bb]
	(;B[dc]
	(;W[gd]
	(;B[gc]
	(;W[id]
	(;B[ka]
	(;W[je]
	(;B[ke]
	(;W[kf]
	(;B[ih]
	(;W[hi]
	(;B[jh]
	(;W[kg]
	(;B[kh]
	(;W[jg]
	(;B[mg]
	(;W[lg]
	(;B[og]
	(;W[lh]
	(;B[mk]
	(;W[mj]
	(;B[sh]
	(;W[si]
	(;B[sg]
	(;W[so]
	(;B[sp]
	(;W[sn]
	(;B[rq]
	(;W[ps]
	(;B[qs]
	(;W[os]
	(;B[nn]
	(;W[no]
	(;B[nm]
	(;W[aj]
	(;B[ak]
	(;W[sc]
	(;B[sd]
	(;W[fg]
	(;B[gg]
	(;W[fi]
	(;B[eh]
	(;W[gk]
	(;B[ik]
	(;W[ij]
	(;B[kk]
	(;W[kj]
	(;B[om]
	(;W[ok]
	(;B[oi]
	(;W[ce]
	(;B[gf]
	(;W[fe]
	(;B[nd]
	(;W[pb]
	(;B[]
	(;W[]
	))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))))`

var ogsGameData GameData = GameData{
	Size:        [2]int{19, 19},
	Komi:        7.5,
	Handicap:    0,
	Winner:      "B",
	Score:       2.5,
	End:         "Scored",
	BlackRank:   "23k",
	WhiteRank:   "19k",
	BlackPlayer: "Spectral-7k",
	WhitePlayer: "Spectral-10k",
	Time:        600,
	Year:        2019,
	Moves:       []string{"Bpd", "Wdd", "Bdp", "Wpp", "Bqn", "Wnq", "Bpj", "Wnc", "Blc", "Wqc", "Bqd", "Wpc", "Bod", "Wnb", "Bme", "Wcn", "Bfq", "Wdj", "Bfc", "Wcf", "Bdb", "Wcc", "Bhd", "Wql", "Bpl", "Wqk", "Bpk", "Wqj", "Bqi", "Wri", "Bqh", "Wrh", "Bqg", "Wqm", "Bpm", "Wpn", "Bpo", "Won", "Bqp", "Woo", "Bqo", "Wpq", "Bqq", "Wrn", "Bro", "Wrm", "Bpr", "Wor", "Bqr", "Whq", "Bbp", "Who", "Bcl", "Wen", "Bel", "Wfj", "Bgl", "Whj", "Bil", "Wjj", "Bkl", "Wlj", "Bml", "Wlm", "Bll", "Wjm", "Bjl", "Whm", "Bhl", "Wfm", "Bfl", "Wkm", "Bmm", "Wmn", "Bim", "Win", "Bgm", "Wgn", "Bhn", "Wio", "Bgo", "Whm", "Bfo", "Wfn", "Bhn", "Weo", "Bep", "Whm", "Bln", "Wlo", "Bhn", "Wgp", "Bfp", "Whm", "Bjn", "Wkn", "Bhn", "Wco", "Bhm", "Wcp", "Bcq", "Wbo", "Bbq", "Wbl", "Bbk", "Wck", "Bdk", "Wcj", "Bbm", "Wbj", "Bal", "Wek", "Bdl", "Woi", "Bpi", "Wnk", "Bnl", "Wjc", "Bkd", "Wlb", "Bkb", "Wkc", "Bmb", "Wld", "Bla", "Wmc", "Blb", "Wjb", "Bmd", "Wle", "Bjd", "Wlf", "Bic", "Wib", "Bhb", "Wja", "Bha", "Wna", "Bia", "Wmf", "Bof", "Wnf", "Boe", "Wie", "Bhe", "Wif", "Bhf", "Wig", "Bhg", "Whh", "Bgh", "Wgi", "Bfh", "Wff", "Bef", "Wee", "Beg", "Wfd", "Bec", "Wcb", "Bcg", "Wbg", "Bch", "Wbh", "Bdf", "Wde", "Bei", "Wej", "Bci", "Wbi", "Bdi", "Wgr", "Bfr", "Wgq", "Bgs", "Whs", "Bfs", "Wir", "Brg", "Wrk", "Brc", "Wrb", "Brd", "Wsb", "Boc", "Wob", "Boh", "Wnh", "Bni", "Wmi", "Boj", "Wnj", "Bng", "Wmh", "Bca", "Wba", "Bda", "Wbb", "Bdc", "Wgd", "Bgc", "Wid", "Bka", "Wje", "Bke", "Wkf", "Bih", "Whi", "Bjh", "Wkg", "Bkh", "Wjg", "Bmg", "Wlg", "Bog", "Wlh", "Bmk", "Wmj", "Bsh", "Wsi", "Bsg", "Wso", "Bsp", "Wsn", "Brq", "Wps", "Bqs", "Wos", "Bnn", "Wno", "Bnm", "Waj", "Bak", "Wsc", "Bsd", "Wfg", "Bgg", "Wfi", "Beh", "Wgk", "Bik", "Wij", "Bkk", "Wkj", "Bom", "Wok", "Boi", "Wce", "Bgf", "Wfe", "Bnd", "Wpb", "B", "W"},
}
var alphaGoGameData GameData = GameData{
	Size:        [2]int{19, 19},
	Komi:        7.5,
	Handicap:    0,
	Winner:      "W",
	Score:       0,
	End:         "Resign",
	BlackRank:   "",
	WhiteRank:   "9p",
	BlackPlayer: "AlphaGo",
	WhitePlayer: "Lee Sedol",
	Time:        7200,
	Year:        2016,
	Moves:       []string{"Bpd", "Wdp", "Bcd", "Wqp", "Bop", "Woq", "Bnq", "Wpq", "Bcn", "Wfq", "Bmp", "Wpo", "Biq", "Wec", "Bhd", "Wcg", "Bed", "Wcj", "Bdc", "Wbp", "Bnc", "Wqi", "Bep", "Weo", "Bdk", "Wfp", "Bck", "Wdj", "Bej", "Wei", "Bfi", "Weh", "Bfh", "Wbj", "Bfk", "Wfg", "Bgg", "Wff", "Bgf", "Wmc", "Bmd", "Wlc", "Bnb", "Wid", "Bhc", "Wjg", "Bpj", "Wpi", "Boj", "Woi", "Bni", "Wnh", "Bmh", "Wng", "Bmg", "Wmi", "Bnj", "Wmf", "Bli", "Wne", "Bnd", "Wmj", "Blf", "Wmk", "Bme", "Wnf", "Blh", "Wqj", "Bkk", "Wik", "Bji", "Wgh", "Bhj", "Wge", "Bhe", "Wfd", "Bfc", "Wki", "Bjj", "Wlj", "Bkh", "Wjh", "Bml", "Wnk", "Bol", "Wok", "Bpk", "Wpl", "Bqk", "Wnl", "Bkj", "Wii", "Brk", "Wom", "Bpg", "Wql", "Bcp", "Wco", "Boe", "Wrl", "Bsk", "Wrj", "Bhg", "Wij", "Bkm", "Wgi", "Bfj", "Wjl", "Bkl", "Wgl", "Bfl", "Wgm", "Bch", "Wee", "Beb", "Wbg", "Bdg", "Weg", "Ben", "Wfo", "Bdf", "Wdh", "Bim", "Whk", "Bbn", "Wif", "Bgd", "Wfe", "Bhf", "Wih", "Bbh", "Wci", "Bho", "Wgo", "Bor", "Wrg", "Bdn", "Wcq", "Bpr", "Wqr", "Brf", "Wqg", "Bqf", "Wjc", "Bgr", "Wsf", "Bse", "Wsg", "Brd", "Wbl", "Bbk", "Wak", "Bcl", "Whn", "Bin", "Whp", "Bfr", "Wer", "Bes", "Wds", "Bah", "Wai", "Bkd", "Wie", "Bkc", "Wkb", "Bgk", "Wib", "Bqh", "Wrh", "Bqs", "Wrs", "Boh", "Wsl", "Bof", "Wsj", "Bni", "Wnj", "Boo", "Wjp"},
}
