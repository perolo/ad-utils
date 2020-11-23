package ad_utils

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/ldap.v2"
	"log"
	"regexp"
	"strings"
)

var l *ldap.Conn

type ADUser struct {
	Uname  string
	Name   string
	Err    string
}

func InitAD(user string, pass string) {

	var err error
	l, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", "ad.global", 389)) //3268
	if err != nil {
		log.Fatal(err)
	}
	//defer l.Close()

	// Reconnect with TLS
	err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Fatal(err)
	}

	// First bind with a read only user
	err = l.Bind(user, pass)
	if err != nil {
		log.Fatal(err)
	}

}

func contains(s []ADUser, e string) bool {
	for _, a := range s {
		if a.Uname == e {
			return true
		}
	}
	return false
}

func Difference(a []ADUser, b map[string]ADUser) []ADUser {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x.Uname] = struct{}{}
	}
	var diff []ADUser
	for _, x := range a {
		if _, found := mb[x.Uname]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
func Difference2(a map[string]ADUser, b []ADUser) []ADUser {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x.Uname] = struct{}{}
	}
	var diff []ADUser
	for _, x := range a {
		if _, found := mb[x.Uname]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}


func GetUnamesInGroup(group string) (users []ADUser, groups []string, eUsers []ADUser) {

	// Search for the given group
//	filter := fmt.Sprintf("(&(objectCategory=group)(cn=%s))", group)
	filter := fmt.Sprintf("(&(objectCategory=group)(cn=%s)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))", group)
	//(!(userAccountControl:1.2.840.113556.1.4.803:=2))
/*	searchRequest := ldap.NewSearchRequest(
		"dc=ad,dc=global",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
//		[]string{"member", "cn", "dn", "sAMAccountName", "name", "distinguishedName"},
//		[]string{"member", "cn", "dn", "sAMAccountName"},
		[]string{"member", "cn", "dn", "samaccountname"},

		nil,
	)*/
	//OU=_InactiveUsersInternal

	sr, err := l.Search(&ldap.SearchRequest{
		BaseDN: "dc=ad,dc=global",
		//		BaseDN: base,
		Scope:  2, // subtree
		Filter: filter,
		Attributes: []string{"member", "cn", "dn"},
		//Attributes: []string{"member", "cn", "dn", "samaccountname"},
	})


//	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	if len(sr.Entries) != 1 {
		fmt.Printf("User does not exist or too many entries returned :  \n")
		var erru ADUser
		erru.Err = "User does not exist or too many entries returned"
		erru.Name = group
		eUsers = append(eUsers, erru)
		//		log.Fatal("User does not exist or too many entries returned")
	} else {
		for _, entry := range sr.Entries[0].Attributes {
			if entry.Name == "member" {
				for _, vall := range entry.Values {
					re := regexp.MustCompile("OU=([^,]+)")
					matches := re.FindAllString(vall, -1)
					for _, aMatch := range matches {
						if aMatch == "OU=Users" {

							vall2 := strings.Replace(vall, "\\,", "", -1)
							re2 := regexp.MustCompile("CN=([^,]+)")
							//re2 := regexp.MustCompile("CN=([^,]+),([^,]+)")
							matches2 := re2.FindStringSubmatch(vall2)
//							fmt.Printf("\"%s\" -> \"%s\"\n", group, matches2[1])
							us, err := GetUserDN(matches2[1])
							if err != nil {
								var erru ADUser
								erru.Err = err.Error()
								erru.Name = matches2[1]
								eUsers = append(eUsers, erru)
							}
//							us, _ := GetUserDN(sw)
							for _, user := range us {
								if !contains(users, user) {
									var  newUser ADUser
									newUser.Name = matches2[1]
									newUser.Uname = user
									users = append(users, newUser)
								}
							}

//							users = append(users, us...)
//						} else if aMatch == "OU=DistributionGroups" || aMatch == "OU=Distribution Groups" || aMatch == "OU=_Distribution Groups" || aMatch == "OU=_Security Groups" || aMatch == "OU=_Divisional" {
						} else if aMatch == "OU=DistributionGroups" || aMatch == "OU=_Security Groups"  {
							re2 := regexp.MustCompile("CN=([^,]+)")
							matches2 := re2.FindStringSubmatch(vall)
							str2 := strings.Replace(matches2[1], "\\", "", -1)
							fmt.Printf("\"%s\" -> \"%s\"\n", group, str2)
							groups = append(groups, str2)
							nusers, ngroups, nerrUsers := GetUnamesInGroup(str2)
							for _, user := range nusers {
								if !contains(users, user.Uname) {
									users = append(users, user)
									//users = append(users, user)
								}
							}
							groups = append(groups, ngroups...)
							eUsers = append(eUsers, nerrUsers...)
						}
					}
				}
			}
		}
	}
	return users, groups, eUsers
}

/*
 * Returns the DN of the object representing the authenticated user.
 */
func GetUserDN(name string) ([]string, error) {
	var uname [] string
	// Find the distinguished name for the user if userPrincipalName used for login
	//		filter := fmt.Sprintf("(userPrincipalName=%s)", ldap.EscapeFilter(name))
	//	filter := fmt.Sprintf("(displayName=%s)", ldap.EscapeFilter(name))
	//	filter := fmt.Sprintf("(anr=%s) and (OU=UsersInternal)", ldap.EscapeFilter(name))
	//filter := fmt.Sprintf("(anr=%s)", ldap.EscapeFilter(name))
	filter := fmt.Sprintf("(&(anr=%s)(objectCategory=person)(objectClass=user)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))", ldap.EscapeFilter(name))
	//filter := fmt.Sprintf("(&(%s)(objectCategory=person)(objectClass=user))", ldap.EscapeFilter(name))

	//base := fmt.Sprintf("dc=ad,dc=global,cn=%s", g)
	result, err := l.Search(&ldap.SearchRequest{
		BaseDN: "dc=ad,dc=global",
		Scope:  2, // subtree
		Filter: filter,
		Attributes: []string{"sAMAccountName"},
	})
	if err != nil {
		return uname, fmt.Errorf("LDAP search failed for user: %v", err)

	}
	if len(result.Entries) == 0 {
		fmt.Printf("Not found in AD: %s \n", name)
		return uname, fmt.Errorf("Not found in AD: %s \n", name)

	} else if len(result.Entries) > 1 {
//		fmt.Printf("More tham one hit for %s : %v \n", name, len(result.Entries))
	}
	for _, e := range result.Entries {
		if strings.Contains(e.DN, "OU=User") {
			uname = append(uname, e.GetAttributeValue("sAMAccountName"))
		} else {
//			fmt.Printf("   Skipping: %s \n", e.GetAttributeValue("sAMAccountName"))
		}
	}

	return uname, nil
}

func CloseAD() {
	l.Close()
}