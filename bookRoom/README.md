# bookRoom

Books a Georgia Tech library study room through LibCal.

## System design

How a request or a run moves through the code, then the choices behind that shape. In the decision charts, a **solid** box is the route this repo takes and a **dashed** box is the alternative it turns down; the edge label is the tradeoff.

### One slot

GT credentials come from the environment. The program checks the configured room and time, then submits that slot on LibCal. It does not search the library for a different room.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  env["GT_USERNAME and GT_PASSWORD"]
  slot["Configured room and time"]
  libcal["LibCal submit"]
  env --> slot --> libcal
  class env,slot,libcal chosen
```

### The same room

The form is the same form most weeks, so the script books one slot. Credentials stay out of the repo. A password in the environment is still a password on the machine.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  classDef rejected fill:#f5f5f5,stroke:#9e9e9e,stroke-width:1px,stroke-dasharray:5 5,color:#616161
  q{What gets booked}
  q -->|"the room and time in the config"| one["One slot"]
  q -->|"whatever else is open"| search["Search the library"]
  cred{Where the password lives}
  cred -->|"the environment"| env["Not in git"]
  cred -->|"next to the script"| git["Committed"]
  class one,env chosen
  class search,git rejected
```

Longer write-up: [aamogh.vercel.app/projects/bookroom](https://aamogh.vercel.app/projects/bookroom.html)
