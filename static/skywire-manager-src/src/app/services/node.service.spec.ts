import { TestBed, inject } from '@angular/core/testing';
import { Node} from '../app.datatypes';
import { NodeService } from './node.service';
import {HttpClient} from '@angular/common/http';

describe('NodeService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [NodeService, HttpClient]
    });
  });

  it('should be created', inject([NodeService], (service: NodeService) => {
    expect(service).toBeTruthy();
  }));
});
