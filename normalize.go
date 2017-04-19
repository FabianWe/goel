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

import "sync"

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

func (tbox *TBox) ApplyPhaseOne(bufferSize int) *PhaseOneNormResult {
	// res := NewPhaseOneNormResult(tbox., bufferSize)
	return nil
}
