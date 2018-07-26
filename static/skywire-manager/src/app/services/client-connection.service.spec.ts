import { TestBed, inject } from '@angular/core/testing';

import { ClientConnectionService } from './client-connection.service';

describe('ClientConnectionService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ClientConnectionService]
    });
  });

  it('should be created', inject([ClientConnectionService], (service: ClientConnectionService) => {
    expect(service).toBeTruthy();
  }));
});
