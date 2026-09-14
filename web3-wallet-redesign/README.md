# Web3 Agent Wallet Redesign

## Context

- Source: `internal/httpapi/ui/index.html`
- Output: a standalone visual prototype; production wallet logic is not connected.
- Audience: mobile-first users managing multi-chain assets through an AI-assisted wallet.

## Assumptions

- The current five product areas remain: Agent, market, trade, pay, and wallet.
- Security-critical flows stay local and must clearly distinguish lock, password change, and destructive reset.
- The first review focuses on visual direction and interaction hierarchy rather than API integration.

## Design direction

“On-chain Cockpit” uses a dark technical canvas, cyan instrumentation accents, compact data typography, restrained violet AI signals, and visible network/security states. The wallet home is intentionally denser than a generic fintech dashboard while keeping touch targets at least 44px.

## Prototype interactions

- Switch network from the top-right selector.
- Show or hide balances.
- Trigger send, receive, swap, bridge, refresh, asset detail, and AI analysis feedback.
- Navigate across all five product areas.
- Open Security Center from the settings button.
- Lock/unlock, change password, and confirm wallet reset.
- Demo password: `Web3demo!`.
