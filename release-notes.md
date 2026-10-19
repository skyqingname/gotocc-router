# 0.2.15+custom.008 — Creative studio sizes and estimated cost

The creative studio now selects image size tiers, aspect ratios, quality and background per model family, following the TokenRouter media capability registration. GPT Image 2 aliases such as `gpt-image-2.5` receive the 1K/2K/4K tiers and five aspect ratios previously limited to the exact `gpt-image-2` name; GPT Image 1, Grok Imagine and Gemini image models receive their own registered pixel tables. The tables live in the root `creative-studio.json`.

The composer shows the estimated price per image and the total for the requested count. Prices come from the model plaza for the selected key's group, using the billed size tier of the submitted pixels and the same effective rate as the plaza. Auto-routing keys, token-priced models and models without plaza prices continue to show usage-based billing.

Submission, task history, storage and billing keep their existing contracts. No database migration is required. Upstream baseline remains Sub2API v0.2.15 with the existing Plus customizations. This candidate has not been published to GitHub.
