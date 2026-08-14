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
		pool = allQuotes()
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
	for _, q := range allQuotes() {
		if q.matches(qlen) {
			out = append(out, q)
		}
	}
	return out
}

func allQuotes() []Quote {
	out := make([]Quote, 0, len(StoicQuotes)+len(FunnyQuotes))
	out = append(out, StoicQuotes...)
	out = append(out, FunnyQuotes...)
	return out
}

// StoicQuotes is the default quote race pack (philosophy, stoic, and wisdom).
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

	// Naval Ravikant & modern wisdom (short)
	{Text: "Play long-term games with long-term people.", Author: "Naval Ravikant"},
	{Text: "Earn with your mind, not your time.", Author: "Naval Ravikant"},
	{Text: "Read what you love until you love to read.", Author: "Naval Ravikant"},
	{Text: "Seek wealth, not money or status.", Author: "Naval Ravikant"},
	{Text: "Impatience with actions, patience with results.", Author: "Naval Ravikant"},
	{Text: "Make something people want.", Author: "Paul Graham"},
	{Text: "The obstacle is the way.", Author: "Ryan Holiday"},
	{Text: "We are what we repeatedly do. Excellence, then, is not an act but a habit.", Author: "Will Durant"},
	{Text: "Simplicity is the ultimate sophistication.", Author: "Leonardo da Vinci"},

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

	{Text: "Desire is a contract you make with yourself to be unhappy until you get what you want.", Author: "Naval Ravikant"},
	{Text: "If you can't see yourself working with someone for life, don't work with them for a day.", Author: "Naval Ravikant"},
	{Text: "A fit body, a calm mind, a house full of love. These things cannot be bought — they must be earned.", Author: "Naval Ravikant"},
	{Text: "You're not going to get rich renting out your time. You must own equity to gain your financial freedom.", Author: "Naval Ravikant"},
	{Text: "All the returns in life, whether in wealth, relationships, or knowledge, come from compound interest.", Author: "Naval Ravikant"},
	{Text: "The way to get startup ideas is not to try to think of startup ideas. It's to look for problems, preferably problems you have yourself.", Author: "Paul Graham"},
	{Text: "It is not the man who has too little, but the man who craves more, that is poor.", Author: "Seneca"},
	{Text: "He who has a why to live can bear almost any how.", Author: "Friedrich Nietzsche"},
	{Text: "The only way to do great work is to love what you do. If you haven't found it yet, keep looking. Don't settle.", Author: "Steve Jobs"},

	// long
	{Text: "You could leave life right now. Let that determine what you do and say and think. If you work at that which is before you, following right reason seriously, vigorously, calmly, without allowing anything else to distract you, you will live happily. And there is no one who can prevent this.", Author: "Marcus Aurelius"},
	{Text: "There is only one way to happiness and that is to cease worrying about things which are beyond the power of our will. Demand not that events should happen as you wish, but wish them to happen as they do happen, and you will go on well.", Author: "Epictetus"},
	{Text: "It is not that we have a short time to live, but that we waste a lot of it. Life is long enough, and a sufficiently generous amount has been given to us for the highest achievements if it were all well invested. But when it is wasted in heedless luxury and spent on no good activity, we are forced at last by death's final constraint to realize that it has passed away before we knew it was passing.", Author: "Seneca"},
	{Text: "At dawn, when you have trouble getting out of bed, tell yourself: I have to go to work as a human being. What do I have to complain of, if I am going to do what I was born for, the things I was brought into the world to do? Or is this what I was created for, to huddle under the blankets and stay warm?", Author: "Marcus Aurelius"},
	{Text: "Remember that it is not he who gives abuse or blows who insults, but the view we take of these things as insulting. When therefore a man gives you abuse, remember to say to yourself that it is your opinion which is the insult.", Author: "Epictetus"},
	{Text: "Let us prepare our minds as if we had come to the very end of life. Let us postpone nothing. Let us balance life's books each day. The one who puts the finishing touches on their life each day is never short of time.", Author: "Seneca"},

	{Text: "How to get rich without getting lucky: seek wealth, not money or status. Wealth is assets that earn while you sleep. Money is how we transfer wealth. Status is your place in the social hierarchy. You're not going to get rich renting out your time. You must own equity to gain your financial freedom.", Author: "Naval Ravikant"},
	{Text: "The modern struggle: lone individuals summoning inhuman willpower, fasting, meditating, and exercising, up against armies of scientists and statisticians weaponizing abundant food, screens, and medicines into dopamine traps.", Author: "Naval Ravikant"},
	{Text: "Arm yourself with specific knowledge, accountability, and leverage. Specific knowledge is knowledge you care about. If you are not fully into it, somebody else who is will outperform you. They do not have to be smarter. They just have to be more focused.", Author: "Naval Ravikant"},
	{Text: "In the long run, optimism is the only realism. Pessimists sound smart, optimists make money. The world is built by optimists who believe the future will be better and then work to make it so.", Author: "Naval Ravikant"},
	{Text: "The best time to plant a tree was twenty years ago. The second best time is now.", Author: "Chinese proverb"},
}

// FunnyQuotes is absurdist fake-wisdom for quote races. Short tautologies plus
// a few medium/long rambles so every length bucket can draw a joke.
var FunnyQuotes = []Quote{
	// short
	{Text: "No one can use you if you are useless.", Author: "Anonymous"},
	{Text: "The longer you don't pee the longer you pee.", Author: "Anonymous"},
	{Text: "If your enemy can predict your next move then don't move.", Author: "Anonymous"},
	{Text: "If you do nothing, nothing can go wrong, except the nothing.", Author: "Anonymous"},
	{Text: "The secret to never being late is to never arrive.", Author: "Anonymous"},
	{Text: "You cannot miss what you never aimed at.", Author: "Anonymous"},
	{Text: "The fastest way to get there is to already be there.", Author: "Anonymous"},
	{Text: "If you are lost, congratulations, you are exactly where you are.", Author: "Anonymous"},
	{Text: "A closed door is just an open wall with extra steps.", Author: "Anonymous"},
	{Text: "If you have two choices, pick the third.", Author: "Anonymous"},
	{Text: "Never start a fight you cannot finish, unless you can start running.", Author: "Anonymous"},
	{Text: "If you wait long enough, the bug becomes a feature.", Author: "Anonymous"},
	{Text: "Don't bite the hand that feeds you. Bite the other one.", Author: "Anonymous"},
	{Text: "The best defense is a good offense, unless you are on fire.", Author: "Anonymous"},
	{Text: "If everything is important, then nothing is, including this sentence.", Author: "Anonymous"},
	{Text: "You cannot drown in a desert, but you can still complain.", Author: "Anonymous"},
	{Text: "Never put all your eggs in one basket. Put them in a fridge.", Author: "Anonymous"},
	{Text: "If the mountain will not come to you, stop yelling at mountains.", Author: "Anonymous"},
	{Text: "A journey of a thousand miles begins with forgetting why you left.", Author: "Anonymous"},
	{Text: "Silence is golden, but typing is louder.", Author: "Anonymous"},

	// medium
	{Text: "If your enemy can predict your next move then don't move. If they predicted the not moving, lie down. If they predicted the lying down, you were never in this fight. You were in a nap.", Author: "Anonymous"},
	{Text: "The longer you don't pee the longer you pee. This is the only natural law that has never been appealed, never been repealed, and never been remembered until it is already too late.", Author: "Anonymous"},
	{Text: "No one can use you if you are useless. This is both a warning and a career strategy. Choose which one after lunch, or never, which is also a choice.", Author: "Anonymous"},
	{Text: "If you have nothing, nobody can take it from you. If you have something, they still might not take it, but now you have to worry. This is why empty pockets sleep so well.", Author: "Anonymous"},

	// long
	{Text: "If your enemy can predict your next move then don't move. Stand very still until the prediction expires. If they predicted the standing still, sit down. If they predicted the sitting down, you have two options: leave, or pretend this was the plan the whole time. Pretending is cheaper. Leaving requires legs. Either way, they cannot use you if you are useless, and they cannot time your bathroom break if you never go. The longer you don't pee the longer you pee. History will not record this. Your bladder will.", Author: "Anonymous"},
}
