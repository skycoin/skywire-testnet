import { TestBed, inject } from '@angular/core/testing';
import { Node} from '../app.datatypes';
import { NodeService } from './node.service';
import {HttpClient} from "@angular/common/http";

describe('NodeService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [NodeService, HttpClient]
    });
  });

  it('should be created', inject([NodeService], (service: NodeService) => {
    expect(service).toBeTruthy();
  }));

  it('test getDefaultNodeLabel', inject([NodeService], (service: NodeService) =>
  {
    let node: Node = { addr: '192.168.1.2', type: '', send_bytes: null, key: null, last_ack_time: null, recv_bytes: null, start_time: null },
        label = NodeService.getDefaultNodeLabel(node);

    expect(label).toEqual('Manager')
  }));
});
