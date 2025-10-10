package api

type AuthService struct {}

type User struct {
    ID       int
    name     string
    email    string
    password string
}

func (as AuthService) Authenticate(login, password string) (*User, error) {
    user, err := as.queryDB(login, password)
    if err != nil {
        user = nil
    }
    return user, err
}

func (as AuthService) NewUser(name, login, password string) (*User, error) {
    if err := validate(login, password); err != nil {
        return nil, err
    }
    user := &User {
        ID: generateID(),
        name: name,
        email: login,
        password: password,
    }
    return user, nil
}

//TODO: make an actual generation
func generateID() int { return 1 }

//TODO: make an actual validation
func validate(login, password string) error {
    return nil
}
