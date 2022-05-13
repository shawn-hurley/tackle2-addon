# tackle2-addon

[![Tackle2 Addon Repository on Quay](https://quay.io/repository/konveyor/tackle2-addon/status "Tackle2 Addon Repository on Quay")](https://quay.io/repository/konveyor/tackle2-addon) [![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html) [![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/konveyor/tackle2-addon/pulls)

Tackle (2nd Generation) addon.

Provides common addon packages for working with:
- Git
- Subversion
- Maven
- SSH credentials

Admin addon provides the the following capabilities specified using
the task _variant_:
- Report mounted volume `Capacity` and `Used`.
  - variant: mount:report
- Delete content of mounted volumes.
  - variant: mount:clean

---

Example tasks:

mount:report
```
{
   "name":"mount-report",
   "state": "Ready",
   "variant":"mount:report",
   "priority": 1,
   "addon": "admin",
   "data": {
     "path": "m2"
   }
}
```

mount:clean
```
{
   "name":"mount-clean",
   "state": "Ready",
   "variant":"mount:clean",
   "priority": 1,
   "policy": "isolated",
   "addon": "admin",
   "data": {
     "path": "m2"
   }
}
```

