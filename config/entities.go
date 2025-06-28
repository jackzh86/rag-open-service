package config

import "strings"

// Helper function to convert slice to map[string]bool
func sliceToMap(words []string) map[string]bool {
	result := make(map[string]bool)
	for _, word := range words {
		result[word] = true
	}
	return result
}

// StopWords contains common words that should be filtered out during entity extraction
var StopWords = sliceToMap([]string{
	// Articles
	"the", "a", "an",

	// Conjunctions
	"and", "or", "but", "nor", "yet", "so",

	// Prepositions
	"in", "on", "at", "to", "for", "of", "with", "by",
	"from", "up", "about", "into", "through", "during",
	"before", "after", "above", "below", "between", "among",

	// Common verbs
	"is", "are", "was", "were", "be", "been", "being",
	"have", "has", "had", "do", "does", "did",
	"will", "would", "could", "should", "may", "might", "can",

	// Pronouns
	"i", "you", "he", "she", "it", "we", "they",
	"me", "him", "her", "us", "them",
	"my", "your", "his", "its", "our", "their",
	"mine", "yours", "hers", "ours", "theirs",

	// Demonstratives
	"this", "that", "these", "those",

	// Common adverbs
	"very", "really", "quite", "rather", "too",
	"just", "only", "even", "still", "already",
	"now", "then", "here", "there", "where", "when",

	// Common adjectives
	"good", "bad", "big", "small", "new", "old",
	"high", "low", "long", "short", "right", "wrong",
	"same", "different", "other", "another", "some", "any",

	// Numbers and time words
	"one", "two", "three", "first", "second", "third",
	"today", "yesterday", "tomorrow",

	// Common words that don't add value
	"thing", "things", "way", "ways", "time", "times",
	"people", "person", "man", "woman", "child", "children",
	"work", "works", "life", "lives", "world", "worlds",
	"day", "days", "year", "years", "month", "months",
	"week", "weeks", "hour", "hours", "minute", "minutes",
})

// GenericTerms contains terms that are too generic to be meaningful entities
var GenericTerms = sliceToMap([]string{
	// Common actions
	"having", "being", "doing", "going", "coming", "getting",
	"making", "taking", "giving", "saying", "telling",
	"knowing", "thinking", "feeling", "seeing", "hearing",

	// Common concepts that are too broad
	"something", "anything", "everything", "nothing",
	"someone", "anyone", "everyone", "noone",
	"somewhere", "anywhere", "everywhere", "nowhere",

	// Common modifiers
	"especially", "particularly", "specifically", "generally",
	"usually", "normally", "typically", "sometimes",
	"often", "always", "never", "rarely",

	// Common connectors
	"however", "therefore", "moreover", "furthermore",
	"meanwhile", "otherwise", "nevertheless", "nonetheless",

	// Common responses
	"yes", "no", "maybe", "perhaps", "probably",
	"definitely", "certainly", "surely", "obviously",

	// Common fillers
	"well", "okay", "right", "sure", "fine",
	"thanks", "thank", "please", "sorry",

	// Common time words
	"whenever", "wherever", "whatever", "whoever",
	"before", "after", "during", "while",
	"once", "twice", "again", "still",

	// Common quantity words
	"many", "much", "few", "little", "several",
	"some", "any", "all", "none", "both",

	// Common comparison words
	"better", "worse", "best", "worst",
	"more", "less", "most", "least",
})

// SignificantWords contains words that are likely to be meaningful entities
var SignificantWords = sliceToMap([]string{
	// Technology terms
	"technology", "software", "hardware", "computer", "internet",
	"programming", "coding", "development", "engineering",
	"algorithm", "database", "network", "system",

	// Business terms
	"business", "company", "startup", "entrepreneur", "investor",
	"funding", "venture", "capital", "market", "product",
	"service", "customer", "revenue", "profit",

	// Academic terms
	"research", "study", "analysis", "theory", "method",
	"experiment", "data", "result", "conclusion",
	"education", "learning", "teaching", "knowledge",

	// Social terms
	"community", "society", "culture", "tradition", "custom",
	"relationship", "family", "friendship", "marriage",
	"parenting", "childhood", "adulthood", "aging",

	// Creative terms
	"art", "music", "literature", "writing", "design",
	"creativity", "innovation", "invention", "discovery",

	// Health terms
	"health", "medicine", "treatment", "therapy", "disease",
	"patient", "doctor", "hospital", "clinic",

	// Environmental terms
	"environment", "nature", "climate", "weather", "pollution",
	"conservation", "sustainability", "renewable", "energy",
})

// IsStopWord checks if a word is a stop word
func IsStopWord(word string) bool {
	return StopWords[strings.ToLower(word)]
}

// IsGenericTerm checks if a term is too generic
func IsGenericTerm(term string) bool {
	return GenericTerms[strings.ToLower(term)]
}

// IsSignificantWord checks if a word is likely to be significant
func IsSignificantWord(word string) bool {
	return SignificantWords[strings.ToLower(word)]
}
