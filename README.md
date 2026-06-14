Shortcuts:

- protobuf messages used as Domain Model
- No separate signing service
- Hardcoded wallet addresses
- No address generation
- Transfer request in 1 go. In prod, transfer would be a multi-step process - validation, queueing, nonce assignment, signing, broadcasting
