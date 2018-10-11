import { TestBed, inject } from '@angular/core/testing';

import { ErrorsnackbarService } from './errorsnackbar.service';

describe('ErrorsnackbarService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ErrorsnackbarService]
    });
  });

  it('should be created', inject([ErrorsnackbarService], (service: ErrorsnackbarService) => {
    expect(service).toBeTruthy();
  }));
});
