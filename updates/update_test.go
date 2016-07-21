package updates

import "testing"

func TestGetTags(t *testing.T) {
	output := `73a324bb5b955c94ba0610d928c4c7a452248160	refs/tags/1.0.0
	13ec47258fba5c1745904837376ab02c22e1068f	refs/tags/1.0.0^{}
	b0953436e570c119e0f15b85761cfffb7ce56906	refs/tags/1.1.0
	0a36da4f66ca7f58e406eed2f639b248fd87173a	refs/tags/1.1.0^{}
	b2487366ef2acf4dbf369dd176b0be6f7dbe5118	refs/tags/1.2.0
	f146c704fba39b7e508436fb4b6cd18f893fde00	refs/tags/1.2.0^{}`

	greatestVersion, err := findLatestVersionTag([]byte(output))
	if err != nil {
		t.Error(err)
	}
	if greatestVersion != "1.2.0" {
		t.Errorf("Wrong version. Got %v, expected %v", greatestVersion, "1.2.0")
	}
}
