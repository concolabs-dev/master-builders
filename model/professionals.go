package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Professional represents a company offering professional services

type Professional struct {
	ID                           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyName                  string             `bson:"company_name" json:"company_name"`
	CompanyType                  string             `bson:"company_type" json:"company_type"`
	CompanyDescription           string             `bson:"company_description" json:"company_description"`
	YearFounded                  int                `bson:"year_founded" json:"year_founded"`
	NumberOfEmployees            int                `bson:"number_of_employees" json:"number_of_employees"`
	Email                        string             `bson:"email" json:"email"`
	TelephoneNumber              string             `bson:"telephone_number" json:"telephone_number"`
	Website                      string             `bson:"website" json:"website"`
	Address                      string             `bson:"address" json:"address"`
	Location                     Location           `bson:"location" json:"location"`
	Specializations              []string           `bson:"specializations" json:"specializations"`
	ServicesOffered              []string           `bson:"services_offered" json:"services_offered"`
	CertificationsAccreditations []string           `bson:"certifications_accreditations" json:"certifications_accreditations"`
	CompanyLogoUrl               string             `bson:"company_logo_url" json:"company_logo_url"`
	CoverImageURL                string             `bson:"cover_image_url" json:"cover_image_url"`
	PID                          string             `bson:"pid" json:"pid"` // Using PID similar to Suppliers
}

// ProfessionalPaymentRecord represents payment records for professionals
type ProfessionalPaymentRecord struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ProfessionalPID string             `bson:"professional_pid" json:"professional_pid"`
	Approved        bool               `bson:"approved" json:"approved"`
	Payments        []Payment          `bson:"payments" json:"payments"`
	Deleted         bool               `bson:"deleted" json:"deleted"`
}
type Project struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Type        string             `bson:"type" json:"type"`
	Location    string             `bson:"location" json:"location"`
	Year        string             `bson:"year" json:"year"`
	Description string             `bson:"description" json:"description"`
	Images      []string           `bson:"images" json:"images"`
	PID         string             `bson:"pid" json:"pid"` // PID of the associated Professional
}
