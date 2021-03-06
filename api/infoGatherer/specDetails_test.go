// Copyright 2015 ThoughtWorks, Inc.

// This file is part of Gauge.

// Gauge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Gauge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Gauge.  If not, see <http://www.gnu.org/licenses/>.

package infoGatherer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/getgauge/gauge/config"
	"github.com/getgauge/gauge/gauge"
	"github.com/getgauge/gauge/util"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

const specDir = "specs"

var _ = Suite(&MySuite{})

var concept1 []byte
var concept2 []byte
var concept3 []byte
var spec1 []byte
var spec2 []byte
var spec3 []byte

type MySuite struct {
	specsDir   string
	projectDir string
}

func (s *MySuite) SetUpTest(c *C) {
	s.projectDir, _ = ioutil.TempDir("_testdata", "gaugeTest")
	s.specsDir, _ = util.CreateDirIn(s.projectDir, specDir)
	config.ProjectRoot = s.projectDir

	s.buildTestData()
}

func (s *MySuite) TearDownTest(c *C) {
	os.RemoveAll(s.projectDir)
}

func (s *MySuite) buildTestData() {
	concept1 = make([]byte, 0)
	concept1 = append(concept1, `# foo bar
* first step with "foo"
* say "hello" to me
* a "final" step
`...)

	concept2 = make([]byte, 0)
	concept2 = append(concept2, `# bar
* first step with "foo"
* say "hello" to me
* a "final" step
`...)

	concept3 = make([]byte, 0)
	concept3 = append(concept3, `# foo bar with <param> having errors
* first step with "foo"
* say <param> to me
* a <final> step
`...)

	spec1 = make([]byte, 0)
	spec1 = append(spec1, `Specification Heading
=====================
Scenario 1
----------
* say hello
* say "hello" to me
`...)

	spec2 = make([]byte, 0)
	spec2 = append(spec2, `Specification Heading
=====================
Scenario 1
----------
* say hello
* say "hello" to me
* say "bye" to me
`...)

	spec3 = make([]byte, 0)
	spec3 = append(spec3, `Specification Heading
=====================
|Col1|Col2|
|----|----|
|Val1|Val2|

Scenario with parse errors
----------
* say hello
* say "hello" to me
* say <bye> to me
`...)
}

func (s *MySuite) TestGetParsedSpecs(c *C) {
	_, err := util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	c.Assert(err, Equals, nil)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{specDir}}

	specFiles := util.FindSpecFilesIn(s.specsDir)
	details := specInfoGatherer.getParsedSpecs(specFiles)

	c.Assert(len(details), Equals, 1)
	c.Assert(details[0].Spec.Heading.Value, Equals, "Specification Heading")
}

func (s *MySuite) TestGetParsedSpecsForInvalidFile(c *C) {
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{specDir}}

	details := specInfoGatherer.getParsedSpecs([]string{"spec1.spec"})

	c.Assert(len(details), Equals, 1)
	c.Assert(len(details[0].Errs), Equals, 1)
	c.Assert(details[0].Errs[0].Message, Equals, "File spec1.spec doesn't exist.")
}

func (s *MySuite) TestGetParsedConcepts(c *C) {
	_, err := util.CreateFileIn(s.specsDir, "concept.cpt", concept1)
	c.Assert(err, Equals, nil)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.projectDir + string(filepath.Separator) + specDir}}

	conceptsMap := specInfoGatherer.getParsedConcepts()

	c.Assert(len(conceptsMap), Equals, 1)
	c.Assert(conceptsMap["foo bar"], NotNil)
	c.Assert(specInfoGatherer.conceptDictionary, NotNil)
}

func (s *MySuite) TestGetParsedStepValues(c *C) {
	steps := []*gauge.Step{
		&gauge.Step{Value: "Step with a {}", LineText: "Step with a <table>", IsConcept: true, HasInlineTable: true},
		&gauge.Step{Value: "A context step", LineText: "A context step", IsConcept: false},
		&gauge.Step{Value: "Say {} to {}", LineText: "Say \"hello\" to \"gauge\"", IsConcept: false,
			Args: []*gauge.StepArg{
				&gauge.StepArg{Name: "first", Value: "hello", ArgType: gauge.Static},
				&gauge.StepArg{Name: "second", Value: "gauge", ArgType: gauge.Static}},
		},
	}

	stepValues := getParsedStepValues(steps)

	c.Assert(len(stepValues), Equals, 2)
	c.Assert(stepValues[0].StepValue, Equals, "A context step")
	c.Assert(stepValues[1].StepValue, Equals, "Say {} to {}")
}

func (s *MySuite) TestInitSpecsCache(c *C) {
	_, err := util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	c.Assert(err, Equals, nil)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(1)

	specInfoGatherer.initSpecsCache()

	c.Assert(len(specInfoGatherer.specsCache.specDetails), Equals, 1)
}

func (s *MySuite) TestInitConceptsCache(c *C) {
	_, err := util.CreateFileIn(s.specsDir, "concept1.cpt", concept1)
	c.Assert(err, Equals, nil)
	_, err = util.CreateFileIn(s.specsDir, "concept2.cpt", concept2)
	c.Assert(err, Equals, nil)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.projectDir + string(filepath.Separator) + specDir}}
	specInfoGatherer.waitGroup.Add(1)

	specInfoGatherer.initConceptsCache()

	c.Assert(len(specInfoGatherer.conceptsCache.concepts), Equals, 2)
}

func (s *MySuite) TestInitStepsCache(c *C) {
	f, _ := util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	f, _ = filepath.Abs(f)
	f1, _ := util.CreateFileIn(s.specsDir, "concept2.cpt", concept2)
	f1, _ = filepath.Abs(f1)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(3)

	specInfoGatherer.initConceptsCache()
	specInfoGatherer.initSpecsCache()
	specInfoGatherer.initStepsCache()
	c.Assert(len(specInfoGatherer.stepsCache.stepValues[f]), Equals, 2)
	c.Assert(len(specInfoGatherer.stepsCache.stepValues[f1]), Equals, 3)

}

func (s *MySuite) TestGetStepsFromCachedSpecs(c *C) {
	var stepsFromSpecsMap = make(map[string][]*gauge.StepValue, 0)
	f, _ := util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	f, _ = filepath.Abs(f)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(3)
	specInfoGatherer.initSpecsCache()

	stepsFromSpecsMap = specInfoGatherer.getStepsFromCachedSpecs()
	c.Assert(len(stepsFromSpecsMap[f]), Equals, 2)
	c.Assert(stepsFromSpecsMap[f][0].StepValue, Equals, "say hello")
	c.Assert(stepsFromSpecsMap[f][1].StepValue, Equals, "say {} to me")
}

func (s *MySuite) TestGetStepsFromCachedConcepts(c *C) {
	var stepsFromConceptsMap = make(map[string][]*gauge.StepValue, 0)
	f, _ := util.CreateFileIn(s.specsDir, "concept1.cpt", concept1)
	f, _ = filepath.Abs(f)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(3)
	specInfoGatherer.initSpecsCache()
	specInfoGatherer.initConceptsCache()

	stepsFromConceptsMap = specInfoGatherer.getStepsFromCachedConcepts()
	c.Assert(len(stepsFromConceptsMap[f]), Equals, 3)
	c.Assert(stepsFromConceptsMap[f][0].StepValue, Equals, "first step with {}")
	c.Assert(stepsFromConceptsMap[f][1].StepValue, Equals, "say {} to me")
	c.Assert(stepsFromConceptsMap[f][2].StepValue, Equals, "a {} step")
}

func (s *MySuite) TestGetAvailableSteps(c *C) {
	var stepValues []*gauge.StepValue
	util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(2)
	specInfoGatherer.initSpecsCache()
	specInfoGatherer.initStepsCache()

	stepValues = specInfoGatherer.Steps()
	c.Assert(len(stepValues), Equals, 2)
	if !hasStep(stepValues, "say hello") {
		c.Fatalf("Step value not found %s", "say hello")
	}
	if !hasStep(stepValues, "say {} to me") {
		c.Fatalf("Step value not found %s", "say {} to me")
	}
}

func (s *MySuite) TestGetAvailableStepsShouldFilterDuplicates(c *C) {
	var stepValues []*gauge.StepValue
	util.CreateFileIn(s.specsDir, "spec2.spec", spec2)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(2)
	specInfoGatherer.initSpecsCache()
	specInfoGatherer.initStepsCache()

	stepValues = specInfoGatherer.Steps()
	c.Assert(len(stepValues), Equals, 2)
	if !hasStep(stepValues, "say hello") {
		c.Fatalf("Step value not found %s", "say hello")
	}
	if !hasStep(stepValues, "say {} to me") {
		c.Fatalf("Step value not found %s", "say {} to me")
	}
}

func hasStep(stepValues []*gauge.StepValue, step string) bool {
	for _, value := range stepValues {
		if value.StepValue == step {
			return true
		}
	}
	return false
}

func (s *MySuite) TestHasSpecForSpecDetail(c *C) {
	c.Assert((&SpecDetail{}).HasSpec(), Equals, false)
	c.Assert((&SpecDetail{Spec: &gauge.Specification{}}).HasSpec(), Equals, false)
	c.Assert((&SpecDetail{Spec: &gauge.Specification{Heading: &gauge.Heading{}}}).HasSpec(), Equals, true)
}

func (s *MySuite) TestGetAvailableSpecDetails(c *C) {
	_, err := util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	c.Assert(err, Equals, nil)
	sig := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}, specsCache: specsCache{specDetails: make(map[string]*SpecDetail)}}
	specFiles := util.FindSpecFilesIn(s.specsDir)
	sig.specsCache.specDetails[specFiles[0]] = &SpecDetail{Spec: &gauge.Specification{Heading: &gauge.Heading{Value: "Specification Heading"}}}

	details := sig.GetAvailableSpecDetails(specFiles)

	c.Assert(len(details), Equals, 1)
	c.Assert(details[0].Spec.Heading.Value, Equals, "Specification Heading")
}

func (s *MySuite) TestGetAvailableSpecDetailsInDefaultDir(c *C) {
	_, err := util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	c.Assert(err, Equals, nil)
	wd, _ := os.Getwd()
	os.Chdir(s.projectDir)
	defer os.Chdir(wd)
	sig := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}, specsCache: specsCache{specDetails: make(map[string]*SpecDetail)}}
	specFiles := util.FindSpecFilesIn(specDir)
	sig.specsCache.specDetails[specFiles[0]] = &SpecDetail{Spec: &gauge.Specification{Heading: &gauge.Heading{Value: "Specification Heading"}}}

	details := sig.GetAvailableSpecDetails([]string{})

	c.Assert(len(details), Equals, 1)
	c.Assert(details[0].Spec.Heading.Value, Equals, "Specification Heading")
}

func (s *MySuite) TestGetAvailableSpecDetailsWithEmptyCache(c *C) {
	_, err := util.CreateFileIn(s.specsDir, "spec1.spec", spec1)
	c.Assert(err, Equals, nil)
	sig := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}

	details := sig.GetAvailableSpecDetails([]string{})

	c.Assert(len(details), Equals, 0)
}

func (s *MySuite) TestParamsForStepFile(c *C) {
	file, _ := util.CreateFileIn(s.specsDir, "spec3.spec", spec3)
	file, _ = filepath.Abs(file)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(2)
	specInfoGatherer.initConceptsCache()
	specInfoGatherer.initSpecsCache()
	specInfoGatherer.initStepsCache()
	specInfoGatherer.initParamsCache()

	staticParams := specInfoGatherer.Params(file, gauge.Static)
	c.Assert(len(staticParams), Equals, 1)
	dynamicParams := specInfoGatherer.Params(file, gauge.Dynamic)
	c.Assert(len(dynamicParams), Equals, 3)
	hasParam := func(param string, list []gauge.StepArg) bool {
		for _, p := range list {
			if p.ArgValue() == param {
				return true
			}
		}
		return false
	}
	if !hasParam("hello", staticParams) {
		c.Errorf(`Param "hello" not found`)
	}
	if !hasParam("bye", dynamicParams) {
		c.Errorf(`Param "bye" not found`)
	}
	if !hasParam("Col1", dynamicParams) {
		c.Errorf(`Param "Col1" not found`)
	}
	if !hasParam("Col2", dynamicParams) {
		c.Errorf(`Param "Col1" not found`)
	}
}

func (s *MySuite) TestParamsForConceptFile(c *C) {
	file, _ := util.CreateFileIn(s.specsDir, "concept3.cpt", concept3)
	file, _ = filepath.Abs(file)
	specInfoGatherer := &SpecInfoGatherer{SpecDirs: []string{s.specsDir}}
	specInfoGatherer.waitGroup.Add(2)
	specInfoGatherer.initConceptsCache()
	specInfoGatherer.initSpecsCache()
	specInfoGatherer.initStepsCache()
	specInfoGatherer.initParamsCache()

	staticParams := specInfoGatherer.Params(file, gauge.Static)
	c.Assert(len(staticParams), Equals, 1)
	dynamicParams := specInfoGatherer.Params(file, gauge.Dynamic)
	c.Assert(len(dynamicParams), Equals, 2)
	hasParam := func(param string, list []gauge.StepArg) bool {
		for _, p := range list {
			if p.ArgValue() == param {
				return true
			}
		}
		return false
	}
	if !hasParam("foo", staticParams) {
		c.Errorf(`Param "foo" not found`)
	}
	if !hasParam("param", dynamicParams) {
		c.Errorf(`Param "param" not found`)
	}
	if !hasParam("final", dynamicParams) {
		c.Errorf(`Param "final" not found`)
	}
}
