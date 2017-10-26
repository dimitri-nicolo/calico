/*
Package users manages and retrieves Users in the OpenStack Identity Service.

Example to List Users

	listOpts := users.ListOpts{
		DomainID: "default",
	}

	allPages, err := users.List(identityClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	allUsers, err := users.ExtractUsers(allPages)
	if err != nil {
		panic(err)
	}

	for _, user := range allUsers {
		fmt.Printf("%+v\n", user)
	}

Example to Create a User

	projectID := "a99e9b4e620e4db09a2dfb6e42a01e66"

	createOpts := users.CreateOpts{
		Name:             "username",
		DomainID:         "default",
		DefaultProjectID: projectID,
		Enabled:          gophercloud.Enabled,
		Password:         "supersecret",
		Extra: map[string]interface{}{
			"email": "username@example.com",
		}
	}

	user, err := users.Create(identityClient, createOpts).Extract()
	if err != nil {
		panic(err)
	}

Example to Update a User

	userID := "0fe36e73809d46aeae6705c39077b1b3"

	updateOpts := users.UpdateOpts{
		Enabled: gophercloud.Disabled,
	}

	user, err := users.Update(identityClient, userID, updateOpts).Extract()
	if err != nil {
		panic(err)
	}

Example to Delete a User

	userID := "0fe36e73809d46aeae6705c39077b1b3"
	err := users.Delete(identityClient, userID).ExtractErr()
	if err != nil {
		panic(err)
	}

Example to List Groups a User Belongs To

	userID := "0fe36e73809d46aeae6705c39077b1b3"

	allPages, err := users.ListGroups(identityClient, userID).AllPages()
	if err != nil {
		panic(err)
	}

	allGroups, err := users.ExtractGroups(allPages)
	if err != nil {
		panic(err)
	}

	for _, group := range allGroups {
		fmt.Printf("%+v\n", group)
	}
*/
package users
