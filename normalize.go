// The MIT License (MIT)

// Copyright (c) 2016, 2017 Fabian Wenzelmann

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package goel

import (
	"sync"
)

// TBoxAccessController is used to concurrently add elements to a normalized
// TBox. It is safe to call the add methods concurrently.
// Note: Don't forget to call the Close method after all add operations are
// done, this will close all channels used to share the memory.
type TBoxAccessController struct {
	Box         *NormalizedTBox
	ciChan      chan *NormalizedCI
	ciLeftChan  chan *NormalizedCILeftEx
	ciRightChan chan *NormalizedCIRightEx
	riChan      chan *NormalizedRI
	wg          *sync.WaitGroup
}

// NewTBoxAccessController returns a new TBoxAccessController.
// It creates the required channels and starts four goroutines to listen
// on those channels. Elements can then be added.
// After creation you must call the Close() method. This will close all
// channels and therefor end the goroutines, it also waits until all elements
// have been properly added.
func NewTBoxAccessController(box *NormalizedTBox) *TBoxAccessController {
	ciChan := make(chan *NormalizedCI, 1)
	ciLeftChan := make(chan *NormalizedCILeftEx, 1)
	ciRightChan := make(chan *NormalizedCIRightEx, 1)
	riChan := make(chan *NormalizedRI, 1)
	var wg sync.WaitGroup
	wg.Add(4)
	// start a go routine that listens on each channel and adds the elements
	// after this each go routine must call wg.Done
	// in Close we wait for the wait group, so close only finsihes when all
	// add operations are Done
	go func() {
		defer wg.Done()
		for ci := range ciChan {
			box.CIs = append(box.CIs, ci)
		}
	}()
	go func() {
		defer wg.Done()
		for ci := range ciLeftChan {
			box.CILeft = append(box.CILeft, ci)
		}
	}()
	go func() {
		defer wg.Done()
		for ci := range ciRightChan {
			box.CIRight = append(box.CIRight, ci)
		}
	}()
	go func() {
		defer wg.Done()
		for ri := range riChan {
			box.RIs = append(box.RIs, ri)
		}
	}()
	return &TBoxAccessController{Box: box, wg: &wg, ciChan: ciChan,
		ciLeftChan: ciLeftChan, ciRightChan: ciRightChan, riChan: riChan}
}

// Close closes all channels and waits until all adding operations are done.
// This method must be called.
// It returns always nil.
// Make sure that this method is called after all AddXXX operations are done,
// no further adds are allowed after you call Close!
func (tbox *TBoxAccessController) Close() error {
	// close all channels
	close(tbox.ciChan)
	close(tbox.ciLeftChan)
	close(tbox.ciRightChan)
	close(tbox.riChan)
	// wait for all to finish
	tbox.wg.Wait()
	return nil
}

// AddCI adds a new NormalizedCI.
func (tbox *TBoxAccessController) AddCI(ci *NormalizedCI) {
	tbox.ciChan <- ci
}

// AddCILeft adds a new NormalizedCILeftEx.
func (tbox *TBoxAccessController) AddCILeft(ci *NormalizedCILeftEx) {
	tbox.ciLeftChan <- ci
}

// AddCIRight adds a new NormalizedCIRightEx.
func (tbox *TBoxAccessController) AddCIRight(ci *NormalizedCIRightEx) {
	tbox.ciRightChan <- ci
}

// AddRI adds a new NormalizedRI.
func (tbox *TBoxAccessController) AddRI(ri *NormalizedRI) {
	tbox.riChan <- ri
}

// PhaseOneNormResult is used in the first phase of the normalization.
// It is used as a way to communicate between the different constraints
// during normalization.
// Don't forget to call Close() once all adding operations are done!
type PhaseOneNormResult struct {
	// TBoxAccessController is used to store the CIs that are already in
	// normal form.
	// Adding elements concurrently is safe.
	// Don't forget to call Close once all Adds are done!
	*TBoxAccessController

	// Components is used to gain information about the unnormalized version.
	// It is also used to generate new names as we need them.
	Components *ELBaseComponents

	// Waiting contains all elements that have not yet been processed.
	Waiting chan *GCIConstraint

	// IntermediateRes contains all elements that can't be processed any further
	// in phase one of the normalization.
	IntermediateRes []*GCIConstraint

	// namesMutex is used to syncronize access to the number of names in the
	// Components object, i.e. we have to generate new names concurrently.
	namesMutex *sync.Mutex
}

func NewPhaseOneNormResult(components *ELBaseComponents, bufferSize int) *PhaseOneNormResult {
	box := NewNormalizedTBox()
	waiting := make(chan *GCIConstraint, bufferSize)
	intermediate := make([]*GCIConstraint, 0)
	return &PhaseOneNormResult{Components: components, namesMutex: &sync.Mutex{},
		TBoxAccessController: NewTBoxAccessController(box), Waiting: waiting,
		IntermediateRes: intermediate}
}

// GetNextName returns the next free NamedConcept id. It is safe for use
// concurrently by different goroutines.
func (phaseRes *PhaseOneNormResult) GetNextName() NamedConcept {
	// lock components mutex
	phaseRes.namesMutex.Lock()
	defer phaseRes.namesMutex.Unlock()
	res := phaseRes.Components.Names
	phaseRes.Components.Names++
	return NamedConcept(res)
}

// TBoxNormalizer is an interface that provides a method to normalize a TBox.
type TBoxNormalizer interface {
	// Normalize normalizes the given TBox.
	Normalize(tbox *TBox) (*NormalizedTBox, error)
}

// DefaultTBoxNormalizer is an implementation of TBoxNormalizer.
type DefaultTBoxNormalizer struct {
	// BufferSize is the number of goroutines that are allowed to run
	// concurrently during normalization.
	BufferSize int
	// PhaseOneRes Is the result created in phase one of the normalization
	// algorithm.
	PhaseOneRes *PhaseOneNormResult
}

func (normalizer *DefaultTBoxNormalizer) Normalize(tbox *TBox) (*NormalizedTBox, error) {
	normalizer.ApplyPhaseOne(tbox)
	return nil, nil
}

// ApplyPhaseOne applies phase one of the normalization.
// If first creates all RIs and then all CIs.
func (normalizer *DefaultTBoxNormalizer) ApplyPhaseOne(tbox *TBox) {
	res := NewPhaseOneNormResult(tbox.Components, normalizer.BufferSize)
	normalizer.PhaseOneRes = res
	normalizer.NormalizeAllRIs(tbox)
	defer res.Close()
}

// NormalizeAllRIs normalizes all RIs.
// It runs NormalizeRI concurrently.
func (normalizer *DefaultTBoxNormalizer) NormalizeAllRIs(tbox *TBox) {
	// add all RIs, take care of buffer size
	riChan := make(chan bool, normalizer.BufferSize)
	// the done channel reports once the errChan has been closed and all values
	// processed
	// a wait group to wait until all started sub-goroutines are done
	var wg sync.WaitGroup
	nextNew := tbox.Components.Roles
	for _, ri := range tbox.RIs {
		// add to the wait group
		wg.Add(1)
		// first write to channel s.t. it can block
		riChan <- true
		// start a go routine that normalizes the RI
		// and reports back once the normalization is done
		go func(ri *RoleInclusion, nextNew uint) {
			// after this function is done:
			// first report that we're done to ri chan
			// next remove 1 from the wait group
			defer func() {
				<-riChan
				wg.Done()
			}()
			normalizer.NormalizeRI(ri, nextNew)
		}(ri, nextNew)
		// an RI of the form r1 o ... rk [= s requires k - 2 new role names if
		// k > 2
		// otherwise it requires 0
		nextNew += uint(IntMax(len(ri.LHS)-2, 0))
	}
	// wait until all normalizations are done
	wg.Wait()
}

// NormalizeRI normalizes a single RI and adds the generated RIs to the
// PhaseOneRes TBox.
func (normalizer *DefaultTBoxNormalizer) NormalizeRI(ri *RoleInclusion, firstNewIndex uint) {
	lhs := ri.LHS
	n := uint(len(lhs))
	switch n {
	case 0:
		// TODO Is this possible?
		normalizer.PhaseOneRes.AddRI(NewNormalizedRI(NoRole, NoRole, ri.RHS))
	case 1:
		normalizer.PhaseOneRes.AddRI(NewNormalizedRI(lhs[0], NoRole, ri.RHS))
	case 2:
		normalizer.PhaseOneRes.AddRI(NewNormalizedRI(lhs[0], lhs[1], ri.RHS))
	default:
		// now we need a loop to add all RIs as given in the algorithm
		// add first one by hand
		// the first one always has the following form:
		// u1 o rk ⊑ s where u1 is a new role
		normalizer.PhaseOneRes.AddRI(NewNormalizedRI(NewRole(firstNewIndex),
			lhs[n-1], ri.RHS))
		var i uint
		for ; i < n-3; i++ {
			normalizer.PhaseOneRes.AddRI(NewNormalizedRI(NewRole(firstNewIndex+i+1),
				lhs[n-2-i], NewRole(firstNewIndex+i)))
		}
		// also add the following RI which is left after the loop:
		// r1 o r2 ⊑ un
		// where un is the last new role
		normalizer.PhaseOneRes.AddRI(NewNormalizedRI(lhs[0], lhs[1], NewRole(firstNewIndex+n-2-1)))
	}
}
