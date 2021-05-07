package syncadgroup

import (
	"flag"
	"fmt"
	"github.com/magiconair/properties"
	"github.com/perolo/ad-utils"
	"github.com/perolo/confluence-prop/client"
	"github.com/perolo/confluence-scripts/utilities"
	excelutils "github.com/perolo/excel-utils"
	"log"
	"path/filepath"
	"time"
)

// or through Decode
type Config struct {
	ConfHost        string `properties:"confhost"`
	User            string `properties:"user"`
	Pass            string `properties:"password"`
	Simple          bool   `properties:"simple"`
	Report          bool   `properties:"report"`
	Limited         bool   `properties:"limited"`
	AdGroup         string `properties:"adgroup"`
	Localgroup      string `properties:"localgroup"`
	File            string `properties:"file"`
	ConfUpload      bool   `properties:"confupload"`
	ConfPage        string `properties:"confluencepage"`
	ConfSpace       string `properties:"confluencespace"`
	ConfAttName     string `properties:"conlfuenceattachment"`
	Bindusername    string `properties:"bindusername"`
	Bindpassword    string `properties:"bindpassword"`
}

func initReport(cfg Config) {
	if cfg.Report {
		excelutils.NewFile()
		excelutils.SetCellFontHeader()
		excelutils.WiteCellln("Introduction")
		excelutils.WiteCellln("Please Do not edit this page!")
		excelutils.WiteCellln("This page is created by the projectreport script: github.com\\perolo\\ad-utils\\SyncADGroup")
		t := time.Now()
		excelutils.WiteCellln("Created by: " + cfg.User + " : " + t.Format(time.RFC3339))
		excelutils.WiteCellln("")
		excelutils.WiteCellln("The Report Function shows:")
		excelutils.WiteCellln("   Ad Names 1- Name and user found in AD Group 1")
		excelutils.WiteCellln("   Ad Names 2- Name and user found in AD Group 2")
		excelutils.WiteCellln("   Not in AD group 1 - Users in the group 2 not found in group 1")
		excelutils.WiteCellln("   Not in AD group 2 - Users in the group 1 not found in group 2")
		excelutils.WiteCellln("   Not in JIRA - Users in the AD not found in the JIRA Group")
		excelutils.WiteCellln("   AD Errors - Internal error when searching for user in AD")
		excelutils.WiteCellln("")
		excelutils.SetCellFontHeader2()
		excelutils.WiteCellln("Group Mapping")
		if cfg.Simple {
			excelutils.WriteColumnsHeaderln([]string{"AD Group 1", "AD Group 2"})
			excelutils.WriteColumnsln([]string{cfg.AdGroup, cfg.Localgroup})
		} else {
			excelutils.WriteColumnsHeaderln([]string{"AD Group 1", "AD Group 2"})
			for _, syn := range GroupSyncs {
					excelutils.WriteColumnsln([]string{syn.AdGroup1, syn.AdGroup2})
			}
		}
		excelutils.WiteCellln("")
		excelutils.SetCellFontHeader2()
		excelutils.WiteCellln("Report")
		excelutils.AutoFilterStart()
		var headers = []string{"Report Function", "AD group 1", "Local Group", "Name", "Uname", "Mail", "Error", "DN"}
		excelutils.WriteColumnsHeaderln(headers)
	}
}

func endReport(cfg Config) error {
	if cfg.Report {
		file := fmt.Sprintf(cfg.File, "-Confluence")
		excelutils.SetColWidth("A", "A", 60)
		excelutils.AutoFilterEnd()
		excelutils.SaveAs(file)
		if cfg.ConfUpload {
			var config = client.ConfluenceConfig{}
			var copt client.OperationOptions
			config.Username = cfg.User
			config.Password = cfg.Pass
			config.URL = cfg.ConfHost
			config.Debug = false
			confluenceClient := client.Client(&config)
			// Intentional override
			copt.Title = "Using AD groups for JIRA/Confluence"
			copt.SpaceKey = "AAAD"
			_, name := filepath.Split(file)
			cfg.ConfAttName = name
			return utilities.AddAttachmentAndUpload(confluenceClient, copt, name, file, "Created by Sync AD group")

		}
	}
	return nil
}

func AdSyncAdGroup(propPtr string) {
	//	propPtr := flag.String("prop", "confluence.properties", "a string")
	flag.Parse()
	p := properties.MustLoadFile(propPtr, properties.ISO_8859_1)
	var cfg Config
	if err := p.Decode(&cfg); err != nil {
		log.Fatal(err)
	}
	toolClient := toollogin(cfg)
	initReport(cfg)
	adutils.InitAD(cfg.Bindusername, cfg.Bindpassword)
	if cfg.Simple {
		SyncGroupInTool(cfg, toolClient)
	} else {
		for _, syn := range GroupSyncs {
				cfg.AdGroup = syn.AdGroup1
				cfg.Localgroup = syn.AdGroup2
				SyncGroupInTool(cfg, toolClient)
		}
	}
	err := endReport(cfg)
	if err != nil {
		panic(err)
	}
	adutils.CloseAD()
}

func toollogin(cfg Config) *client.ConfluenceClient {
	var config = client.ConfluenceConfig{}
	config.Username = cfg.User
	config.Password = cfg.Pass
	config.URL = cfg.ConfHost
	config.Debug = false
	return client.Client(&config)
}

func SyncGroupInTool(cfg Config, client *client.ConfluenceClient) {
	var toolGroupMemberNames map[string]adutils.ADUser
	fmt.Printf("\n")
	fmt.Printf("SyncGroup AdGroup: %s LocalGroup: %s \n", cfg.AdGroup, cfg.Localgroup)
	fmt.Printf("\n")
	var adUnames1 []adutils.ADUser
	var adUnames2 []adutils.ADUser
	if cfg.AdGroup != "" {
		adUnames1, _ = adutils.GetUnamesInGroup(cfg.AdGroup)
		fmt.Printf("adUnames1(%v)\n", len(adUnames1))
	}
	if cfg.Report {
		if !cfg.Limited {
			for _, adu := range adUnames1 {
				var row = []string{"AD Names  1 ", cfg.AdGroup, cfg.Localgroup, adu.Name, adu.Uname, adu.Mail, adu.Err, adu.DN}
				excelutils.WriteColumnsln(row)
			}
		}
	}
	if cfg.AdGroup != "" {
		adUnames2, _ = adutils.GetUnamesInGroup(cfg.Localgroup)
		fmt.Printf("adUnames1(%v)\n", len(adUnames1))
	}
	if cfg.Report {
		if !cfg.Limited {
			for _, adu2 := range adUnames2 {
				var row = []string{"AD Names  2 ", cfg.AdGroup, cfg.Localgroup, adu2.Name, adu2.Uname, adu2.Mail, adu2.Err, adu2.DN}
				excelutils.WriteColumnsln(row)
				toolGroupMemberNames[adu2.Uname] = adu2
			}
		}
	}
	if cfg.Localgroup != "" && cfg.AdGroup != "" {
		notInTool := adutils.Difference(adUnames1, toolGroupMemberNames)
		if len(notInTool) == 0 {
			fmt.Printf("Not In Tool(%v)\n", len(notInTool))
		} else {
			fmt.Printf("Not In Tool(%v) ", len(notInTool))
			for _, nit := range notInTool {
				fmt.Printf("%s, ", nit.Uname)
			}
			fmt.Printf("\n")
		}
		if cfg.Report {
			for _, nji := range notInTool {
				var row = []string{"AD group users not found in Tool user group", cfg.AdGroup, cfg.Localgroup, nji.Name, nji.Uname, nji.Mail, nji.Err, nji.DN}
				excelutils.WriteColumnsln(row)
			}
		}
		notInAD := adutils.Difference2(toolGroupMemberNames, adUnames1)
		if len(notInAD) == 0 {
			fmt.Printf("notInAD(%v)\n", len(notInAD))
		} else {
			fmt.Printf("notInAD(%v) ", len(notInAD))
			for _, nit := range notInAD {
				fmt.Printf("%s, ", nit.Uname)
			}
			fmt.Printf("\n")
		}
		if cfg.Report {
			for _, nad := range notInAD {
				if nad.DN == "" {

					dn, err := adutils.GetActiveUserDN(nad.Uname)
					if err == nil {
						nad.DN = dn.DN
						nad.Mail = dn.Mail
					} else {
						udn, err := adutils.GetAllUserDN(nad.Uname)
						if err == nil {
							nad.DN = udn.DN
							nad.Mail = udn.Mail
							nad.Err = "Deactivated"
						} else {
							edn, err := adutils.GetAllEmailDN(nad.Mail)
							if err == nil {
								nad.DN = edn[0].DN
								nad.Mail = edn[0].Mail
								nad.Err = edn[0].Err
								for _, ldn := range edn {
									var row2 = []string{"Tool user group member not found in AD group (multiple?)", cfg.AdGroup, cfg.Localgroup, nad.Name, nad.Uname, ldn.Mail, ldn.Err, ldn.DN}
									excelutils.WriteColumnsln(row2)
								}
							} else {

								nad.Err = err.Error()
							}
						}
					}

				}
				var row = []string{"Tool user group member not found in AD group", cfg.AdGroup, cfg.Localgroup, nad.Name, nad.Uname, nad.Mail, nad.Err, nad.DN}
				excelutils.WriteColumnsln(row)
			}
		}
	}
}
func getUnamesInToolGroup(theClient *client.ConfluenceClient, localgroup string) map[string]adutils.ADUser {
	groupMemberNames := make(map[string]adutils.ADUser)
	cont := true
	start := 0
	max := 50
	for cont {
		groupMembers, err := theClient.GetGroupMembers(localgroup, &client.GetGroupMembersOptions{StartAt: start, MaxResults: max, ShowBasicDetails: true})
		if err != nil {
			panic(err)
		}
		for _, member := range groupMembers.Users {
			if _, ok := groupMemberNames[member.Name]; !ok {
				var newUser adutils.ADUser
				newUser.Uname = member.Name
				newUser.Name = member.FullName
				newUser.Mail = member.Email
				groupMemberNames[member.Name] = newUser
			}
		}
		if len(groupMembers.Users) != max {
			cont = false
		} else {
			start = start + max
		}
	}
	return groupMemberNames
}