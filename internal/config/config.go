package config

import (
	"log"

	"github.com/spf13/viper"
)

type Calendar struct {
	Name        string `mapstructure:"name" json:"name"`
	WeeksPast   int    `mapstructure:"weeks_past" json:"weeks_past"`
	WeeksFuture int    `mapstructure:"weeks_future" json:"weeks_future"`
	BaseURL     string `mapstructure:"base_url" json:"base_url"`
	Username    string `mapstructure:"username" json:"username"`
	EncPassword string `mapstructure:"enc_password" json:"enc_password"`
	Token       string `mapstructure:"token" json:"token"`
}

type Config struct {
	Calendars []Calendar `mapstructure:"calendars" json:"calendars"`
}

func LoadConfig() Config {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	var appConfig Config

	err := viper.Unmarshal(&appConfig)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	return appConfig
}
