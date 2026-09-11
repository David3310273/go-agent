package milvus

type Config struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	ClusterName string `json:"clusterName"`
	Endpoint    string `json:"endpoint"`
	APIKey      string `json:"apiKey"`
}
