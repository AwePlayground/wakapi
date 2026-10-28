import unittest

from refresh_ai_pricing import normalize_litellm, normalize_models_dev, normalize_openrouter


class NormalizePricingTests(unittest.TestCase):
    def test_litellm_per_token_rates(self):
        result = normalize_litellm({"opus": {
            "input_cost_per_token": 0.000005,
            "cache_read_input_token_cost": 0.0000005,
            "output_cost_per_token": 0.000025,
        }})
        self.assertEqual(result["opus"]["cached_input"], 0.0000005)
        self.assertEqual(result["opus"]["source"], "litellm")

    def test_models_dev_per_million_rates(self):
        result = normalize_models_dev({"anthropic": {"models": {"opus": {
            "cost": {"input": 5, "cache_read": 0.5, "output": 25}
        }}}})
        self.assertEqual(result["anthropic/opus"]["input"], 0.000005)
        self.assertEqual(result["anthropic/opus"]["cached_input"], 0.0000005)

    def test_openrouter_per_token_rates(self):
        result = normalize_openrouter({"data": [{
            "id": "anthropic/opus",
            "pricing": {"prompt": "0.000005", "input_cache_read": "0.0000005", "completion": "0.000025"},
        }]})
        self.assertEqual(result["anthropic/opus"]["output"], 0.000025)


if __name__ == "__main__":
    unittest.main()
