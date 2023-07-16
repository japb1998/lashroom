package client

type Store interface {
	GetClientsByCreator(string) ([]ClientDto, error)
	UpdateUser(createdBy string, userId string, client ClientDto) (ClientDto, error)
	CreateClient(ClientDto) (ClientDto, error)
	DeleteClient(createdBy, id string) error
	GetClientById(createdBy, id string) (*ClientDto, error)
	GetClientWithFilters(createdBy string, clientDto ClientDto) ([]ClientDto, error)
}

type ClientService struct {
	Store Store
}

type ClientDto struct {
	CreatedBy    string  `json:"createdBy"`
	Id           *string `json:"id"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	ClientName   string  `json:"clientName"`
	CreatedAt    string  `json:"createdAt"`
	LastUpdateAt string  `json:"lastUpdateAt"`
	Description  string  `json:"description"`
}

func NewClientService(s Store) *ClientService {

	return &ClientService{
		Store: s,
	}
}
func (c ClientService) GetClientsByCreator(createdBy string) ([]ClientDto, error) {
	clientList, err := c.Store.GetClientsByCreator(createdBy)

	if err != nil {
		return nil, err
	}

	return clientList, err
}

func (c ClientService) UpdateUser(createdBy string, clientId string, client ClientDto) (ClientDto, error) {
	clientDto, err := c.Store.UpdateUser(createdBy, clientId, client)

	if err != nil {
		return ClientDto{}, err
	}

	return clientDto, nil
}

func (c ClientService) CreateClient(client ClientDto) (ClientDto, error) {
	clientDto, err := c.Store.CreateClient(client)

	if err != nil {
		return ClientDto{}, err
	}

	return clientDto, nil
}

func (c ClientService) DeleteClient(createdBy, id string) error {
	err := c.Store.DeleteClient(createdBy, id)

	if err != nil {
		return err
	}

	return nil
}

func (c ClientService) GetClientById(createdBy, id string) (*ClientDto, error) {
	client, err := c.Store.GetClientById(createdBy, id)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c ClientService) GetClientWithFilters(createdBy string, clientDto ClientDto) ([]ClientDto, error) {
	clientList, err := c.Store.GetClientWithFilters(createdBy, clientDto)

	if err != nil {
		return nil, err
	}

	return clientList, nil
}
