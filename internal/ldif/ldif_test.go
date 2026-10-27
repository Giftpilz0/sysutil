package ldif

import (
	"bytes"
	"strings"
	"testing"
)

const sample = `version: 1

# a comment
dn: uid=alice,ou=people,dc=example,dc=org
uid: alice
mail: alice@example.org
mail: a@example.org
description: a long
  folded line

dn: uid=bob,ou=people,dc=example,dc=org
uid: bob
cn:: QsO2Yg==
`

func TestLDIFToCSV(t *testing.T) {
	var out bytes.Buffer

	n, err := LDIFToCSV(strings.NewReader(sample), &out, ';')
	if err != nil {
		t.Fatal(err)
	}

	if n != 2 {
		t.Errorf("got %d entries, want 2", n)
	}

	want := "dn;uid;mail;description;cn\n" +
		"uid=alice,ou=people,dc=example,dc=org;alice;[alice@example.org][a@example.org];a long folded line;\n" +
		"uid=bob,ou=people,dc=example,dc=org;bob;;;Böb\n"
	if out.String() != want {
		t.Errorf("unexpected CSV:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestCSVToLDIF(t *testing.T) {
	in := "dn;mail\n" +
		"uid=alice,dc=example;[a@example.org][b@example.org]\n" +
		"uid=bob,dc=example;\n" +
		"uid=carol,dc=example; carol@example.org\n"

	var out bytes.Buffer

	n, err := CSVToLDIF(strings.NewReader(in), &out, ';', Modify{Attribute: "mail", ChangeType: "replace"})
	if err != nil {
		t.Fatal(err)
	}

	if n != 2 {
		t.Errorf("got %d records, want 2", n)
	}

	want := "dn: uid=alice,dc=example\nchangetype: modify\nreplace: mail\nmail: a@example.org\nmail: b@example.org\n-\n\n" +
		"dn: uid=carol,dc=example\nchangetype: modify\nreplace: mail\nmail:: IGNhcm9sQGV4YW1wbGUub3Jn\n-\n\n"
	if out.String() != want {
		t.Errorf("unexpected LDIF:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestCSVToLDIFDelete(t *testing.T) {
	var out bytes.Buffer

	_, err := CSVToLDIF(strings.NewReader("DN,uid\n\"uid=a,dc=x\",a\n"), &out, ',', Modify{Attribute: "uid", ChangeType: "delete"})
	if err != nil {
		t.Fatal(err)
	}

	if want := "dn: uid=a,dc=x\nchangetype: modify\ndelete: uid\n-\n\n"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestCSVToLDIFErrors(t *testing.T) {
	cases := map[string]struct {
		csv string
		mod Modify
	}{
		"bad change type": {"dn,uid\n", Modify{Attribute: "uid", ChangeType: "nope"}},
		"missing column":  {"dn,uid\n", Modify{Attribute: "mail", ChangeType: "add"}},
		"empty":           {"", Modify{Attribute: "uid", ChangeType: "add"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := CSVToLDIF(strings.NewReader(tc.csv), &bytes.Buffer{}, ',', tc.mod); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseSeparator(t *testing.T) {
	for in, want := range map[string]rune{";": ';', ",": ',', "tab": '\t', `\t`: '\t', "|": '|'} {
		got, err := ParseSeparator(in)
		if err != nil || got != want {
			t.Errorf("ParseSeparator(%q) = %q, %v", in, got, err)
		}
	}

	for _, in := range []string{"", ";;", `"`, "\n"} {
		if _, err := ParseSeparator(in); err == nil {
			t.Errorf("ParseSeparator(%q) should fail", in)
		}
	}
}
