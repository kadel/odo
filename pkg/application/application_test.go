package application

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/redhat-developer/ocdev/mocks"
)

func TestCreate(t *testing.T) {
	type args struct {
		applicationName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "create new application",
			args: args{
				applicationName: "appname",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create moc OpenShiftClient
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockOpenShiftClient := mocks.NewMockOpenShiftClient(mockCtrl)

			mockOpenShiftClient.EXPECT().GetCurrentProjectName().Return("project", nil).Times(1)

			// have separate config file for each test
			tempConfigFile, err := ioutil.TempFile("", "ocdevconfig")
			if err != nil {
				t.Fatal(err)
			}
			defer tempConfigFile.Close()
			os.Setenv("OCDEVCONFIG", tempConfigFile.Name())

			if err := Create(tt.args.applicationName, mockOpenShiftClient); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}

			content, err := ioutil.ReadFile(tempConfigFile.Name())

			fmt.Printf("%s", string(content[:]))

		})
	}
}
