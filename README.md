# ppp-profiler

This service listens to realtime updates from [mhist](https://github.com/codeuniversity/ppp-mhist/), which are evaluated with scripts that are sent through a http-endpoint. The output is a state of a card that is then shown in some kind of frontend, that receives the updated card state via websocket.

## endpoints

- `/` endpoint that listens for websocket connections
- `/profiles` endpoint for profiles
  - `POST` creates a new profile, according to the [ProfileDefinition](https://github.com/codeuniversity/ppp-profiler/blob/53c7d06557d873dd1a878704c0e4ba80e373e65b/profile_definition.go#L7-L14) you sent
  - `GET` returns all profiles currently handled by the profiler:
```
[
    {
        "definition": {
            "eval_script": <string>,
            "filter": {
                "names": <string[]>,
            },
            "id": <string>,
            "is_local": <boolean>,
            "library_id": <string>
        },
        "display": {
            <current display state - see Profile struct >
        },
        "store": {
            <current profile store state - see Profile struct>
        }
    },
...
]
```
- `profiles/{profile_definition_id}` - endpoint for a specific profile
  - `PUT` update [ProfileDefinition](https://github.com/codeuniversity/ppp-profiler/blob/53c7d06557d873dd1a878704c0e4ba80e373e65b/profile_definition.go#L7-L14) 
  - `DELETE`deletes profile
- `/meta` serves [meta data from mhist](https://github.com/codeuniversity/ppp-mhist/blob/826e9d59d8a6289f45fb671c25489bff55afc7e6/disk_meta.go#L26-L30):
```
[
    {
        "name": "temp",
        "type": 1
    },
  ...
]
 ```

## scripts
Scripts you send to this service are javascript that is interpreted by [otto](https://github.com/robertkrimen/otto).
For examples look into the `example_scripts` directory.
They are called once for every message from mhist, that passes the optionally defined filter in the ProfileDefinition (some scripts might only make sense for certain types of messages).
The following functions are defined through this service:

### `set(key: string, value: any)`
  store a value under a certain key, to be used at any later call

### `get(key: string, [defaultValue: any])` 
  get a value stored under a certain key, that was set with `set()`. The default value is used, when no value is found. If no default value is given and no value is found `undefined` is returned.

### `title(text: string)` 
  defines the profile title, that is supposed to be shown in a frontend. If this function is not called, the profile will not have a title.
  
### `description(text: string)` 
  same as `title`, but with profile description
  
### `action(text: string)` 
  same as `title` and `description`, but with profile action
