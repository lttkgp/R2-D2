# R2-D2

R2-D2 is [C-3PO](https://github.com/lttkgp/C-3PO)'s beloved sidekick, helping with fetching and managing all the Facebook group data for Listen To This KGP. He saves a copy of the Facebook public data (no user information - mostly because he can't get it even if he wanted to) and keeps it in sync with C-3PO.

## Getting started
### Prerequisites
- Docker
- <details>
    <summary> Facebook Graph API Credentials </summary>

    You will need 'User access tokens' to work with the Graph API. You can find more information here: [Graph API documentation](https://developers.facebook.com/docs/graph-api/overview#step2).

    As explained in the link above, create a new Facebook app (My Apps -> Add a new app) and generate user access tokens through the Graph API explorer.
  </details>

### Setting up
- Create a `.env` file, using the `.env.template` file as reference.
  ```sh
  cp .env.template .env
  ```
  Fill all the fields using the credentials created as part of the pre-requisites.

### Running the scheduler
Run the scheduler with
```sh
docker-compose up
```

## Contributing
Contributions are always welcome. Your contributions could either be creating new features, fixing bugs or improving documentation and examples. Find more detailed information in [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## License
[MIT](LICENSE)
