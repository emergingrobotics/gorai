# Configuration Guide

GoRAI robots are configured via JSON.

## Example Configuration

```json
{
  "components": [
    {
      "name": "left_motor",
      "type": "motor",
      "model": "gpio",
      "config": {
        "pin": 18,
        "frequency": 1000
      }
    }
  ]
}
```

## Hot Reconfiguration

Configuration can be updated at runtime without restarting.

*More content coming soon*
