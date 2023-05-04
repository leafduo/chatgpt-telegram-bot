package main

import (
	"log"
	"sync"

	tokenizer "github.com/samber/go-gpt-3-encoder"
)

var encoder *tokenizer.Encoder
var once sync.Once

func CountToken(msg string) (int, error) {
	once.Do(func() {
		var err error
		encoder, err = tokenizer.NewEncoder()
		if err != nil {
			log.Fatal(err)
		}
	})

	/**
	The exact algorithm for counting tokens has been documented at
	https://github.com/openai/openai-cookbook/blob/main/examples/How_to_count_tokens_with_tiktoken.ipynb. However, we
	are not using that algorithm in this project because no Go library currently implements the ChatGPT algorithm. The
	Go library github.com/samber/go-gpt-3-encoder does implement the encoding used by GPT-3(p50k_base), but not
	ChatGPT(cl100k_base). Additionally, counting tokens is not a critical part of this project; we only require a rough
	estimation of the token count.

	Based on my naïve experiments, the token count is not significantly different when English text is tokenized.
	However, there is a significant difference when tokenizing Chinese or Japanese text. cl100k_base generates far fewer
	tokens than p50k_base when tokenizing Chinese or Japanese. Most Chinese characters are counted as 1 token in
	cl100k_base, whereas in p50k_base, they are mostly counted as 2 tokens.
	*/

	encoded, err := encoder.Encode(msg)
	if err != nil {
		return 0, err
	}

	// 4 is the number of tokens added by the encoder
	return len(encoded) + 4, nil
}
