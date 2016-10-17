# RLTop - Redis List Top


## Descritption

A simple appliciation developed to monitor the backlog of items in one or many Redis lists on one or many redis hosts.  This mostly useful for monitoring Redis in the context of using it as a buffer for Logstash.

Please note this project is still considered in beta, therefore the code isn't necessarily optimal and some bugs may be present.

## Building

Run `make build`

## Configuration format:

```
{
  "redis_hosts": 
    {
      "[HOSTNAME]:[PORT]": [
        "[LIST_1]",
        "[LIST_2]"
      ]
    },
    {
      "[HOSTNAME2]:[PORT]": [
        "[LIST_1]",
        "[LIST_2]"
      ]
    }
}
```


## Running

### Locally/Testing

- In a seperate terminal window, run the script to push test data into the test lists: `redis_test_push.sh`
- In a seperate terminal window, run the script to pop test data from the test lists: `redis_test_pop.sh` 
- In a seperate terminal window, run the application: `./rltop -c config.sample.json -r 2`

### With remote instance

- Simply copy the `config.sample.json` config file and add in necessary hostnames/ports along with the proper redis list names

## Feature Requests/Bugs/Improvements

Please create an issue for this and I'll take a look at it when time permits.

## Author

Alain Lefebvre <hartfordfive 'at' gmail.com>

## License

Covered under the MIT license.