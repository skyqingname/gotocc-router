# Design: NewAPI unified video compatibility

## Decision

Use the existing OpenAI-compatible video handler for the complete NewAPI task lifecycle.

- `/videos` and `/videos/generations` both enter the OpenAI handler when the selected API key group is OpenAI.
- The outbound path normalizer maps generation aliases onto `/v1/videos`, `/v1/videos/:task_id`, and `/v1/videos/:task_id/content`.
- The request body and response remain transparent; only the HTTP path and upstream credential are controlled by the gateway.
- Channel request validation uses the same `video` billing-mode vocabulary as the service and frontend, allowing the unified channel's per-second rules to be updated normally.
- Real Grok groups continue to use the native Grok media handlers.

## Safety

The group platform, not a model-name prefix, selects the local handler. OpenAI groups never receive the native Grok credential path, and client credentials are not forwarded upstream. Provider-specific edit and extension paths remain Grok-only. Non-video endpoints are unchanged.

## Compatibility

No database migration or new configuration field is required. Infinite Canvas can keep its current `grok-imagine-video` alias while the production group remains a single OpenAI-compatible NewAPI channel.
