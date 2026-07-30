package words

import (
	"math/rand/v2"
	"strings"
	"unicode/utf8"
)

// QuoteLen buckets quotes by character count (Monkeytype-ish).
type QuoteLen int

const (
	QuoteShort QuoteLen = iota
	QuoteMedium
	QuoteLong
)

func (q QuoteLen) String() string {
	switch q {
	case QuoteMedium:
		return "medium"
	case QuoteLong:
		return "long"
	default:
		return "short"
	}
}

// Quote is a passage to type as a race.
type Quote struct {
	Text   string
	Author string
}

// Words splits the quote into typing tokens (whitespace-separated).
func (q Quote) Words() []string {
	return strings.Fields(q.Text)
}

func (q Quote) runeLen() int {
	return utf8.RuneCountInString(q.Text)
}

func (q Quote) matches(qlen QuoteLen) bool {
	n := q.runeLen()
	switch qlen {
	case QuoteShort:
		return n <= 110
	case QuoteMedium:
		return n > 110 && n <= 240
	case QuoteLong:
		return n > 240
	default:
		return true
	}
}

// PickQuote chooses a quote for the length bucket. seed 0 = random.
func PickQuote(qlen QuoteLen, seed uint64) Quote {
	pool := quotesFor(qlen)
	if len(pool) == 0 {
		pool = StoicQuotes
	}
	var r *rand.Rand
	if seed == 0 {
		r = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	} else {
		r = rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	}
	return pool[r.IntN(len(pool))]
}

func quotesFor(qlen QuoteLen) []Quote {
	var out []Quote
	for _, q := range StoicQuotes {
		if q.matches(qlen) {
			out = append(out, q)
		}
	}
	return out
}

// StoicQuotes is the default quote race pack (philosophy / stoic).
var StoicQuotes = []Quote{
	// short
	{Text: "You have power over your mind not outside events. Realize this, and you will find strength.", Author: "Marcus Aurelius"},
	{Text: "Waste no more time arguing about what a good man should be. Be one.", Author: "Marcus Aurelius"},
	{Text: "It is not what happens to you, but how you react to it that matters.", Author: "Epictetus"},
	{Text: "We suffer more often in imagination than in reality.", Author: "Seneca"},
	{Text: "The impediment to action advances action. What stands in the way becomes the way.", Author: "Marcus Aurelius"},
	{Text: "No man is free who is not master of himself.", Author: "Epictetus"},
	{Text: "Luck is what happens when preparation meets opportunity.", Author: "Seneca"},
	{Text: "If it is not right, do not do it. If it is not true, do not say it.", Author: "Marcus Aurelius"},
	{Text: "First say to yourself what you would be; and then do what you have to do.", Author: "Epictetus"},
	{Text: "Begin at once to live, and count each separate day as a separate life.", Author: "Seneca"},
	{Text: "The best revenge is not to be like your enemy.", Author: "Marcus Aurelius"},
	{Text: "He who fears death will never do anything worthy of a living man.", Author: "Seneca"},
	{Text: "Make the best use of what is in your power, and take the rest as it happens.", Author: "Epictetus"},
	{Text: "How long are you going to wait before you demand the best for yourself?", Author: "Epictetus"},
	{Text: "Difficulties strengthen the mind, as labor does the body.", Author: "Seneca"},

	// medium
	{Text: "When you arise in the morning, think of what a privilege it is to be alive, to think, to enjoy, to love. Then set your hands to what is yours to do, and leave the rest.", Author: "Marcus Aurelius"},
	{Text: "Men are disturbed not by things, but by the views which they take of them. Therefore when we are hindered or disturbed or grieved, let us never blame others, but ourselves.", Author: "Epictetus"},
	{Text: "True happiness is to enjoy the present, without anxious dependence upon the future, not to amuse ourselves with either hopes or fears but to rest satisfied with what we have.", Author: "Seneca"},
	{Text: "Never let the future disturb you. You will meet it, if you have to, with the same weapons of reason which today arm you against the present.", Author: "Marcus Aurelius"},
	{Text: "Freedom is the only worthy goal in life. It is won by disregarding things that lie beyond our control. Stop aspiring to be anyone other than your own best self.", Author: "Epictetus"},
	{Text: "Life is long if you know how to use it. We are not given a short life but we make it short, and we are not ill-supplied but wasteful of it.", Author: "Seneca"},
	{Text: "Accept the things to which fate binds you, and love the people with whom fate brings you together, but do so with all your heart.", Author: "Marcus Aurelius"},
	{Text: "Remember that you are an actor in a play, which is as the author wants it to be. Your business is to act well the part that is given; selection of the part is another's.", Author: "Epictetus"},
	{Text: "As long as you live, keep learning how to live. It takes a whole life to learn how to live, and a whole life to learn how to die.", Author: "Seneca"},
	{Text: "Do not act as if you were going to live ten thousand years. Death hangs over you. While you live, while it is in your power, be good.", Author: "Marcus Aurelius"},

	// long
	{Text: "You could leave life right now. Let that determine what you do and say and think. If you work at that which is before you, following right reason seriously, vigorously, calmly, without allowing anything else to distract you, you will live happily. And there is no one who can prevent this.", Author: "Marcus Aurelius"},
	{Text: "There is only one way to happiness and that is to cease worrying about things which are beyond the power of our will. Demand not that events should happen as you wish, but wish them to happen as they do happen, and you will go on well.", Author: "Epictetus"},
	{Text: "It is not that we have a short time to live, but that we waste a lot of it. Life is long enough, and a sufficiently generous amount has been given to us for the highest achievements if it were all well invested. But when it is wasted in heedless luxury and spent on no good activity, we are forced at last by death's final constraint to realize that it has passed away before we knew it was passing.", Author: "Seneca"},
	{Text: "At dawn, when you have trouble getting out of bed, tell yourself: I have to go to work as a human being. What do I have to complain of, if I am going to do what I was born for, the things I was brought into the world to do? Or is this what I was created for, to huddle under the blankets and stay warm?", Author: "Marcus Aurelius"},
	{Text: "Remember that it is not he who gives abuse or blows who insults, but the view we take of these things as insulting. When therefore a man gives you abuse, remember to say to yourself that it is your opinion which is the insult.", Author: "Epictetus"},
	{Text: "Let us prepare our minds as if we had come to the very end of life. Let us postpone nothing. Let us balance life's books each day. The one who puts the finishing touches on their life each day is never short of time.", Author: "Seneca"},
}
