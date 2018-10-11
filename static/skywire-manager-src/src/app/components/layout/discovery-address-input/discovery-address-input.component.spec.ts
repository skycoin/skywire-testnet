import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { DiscoveryAddressInputComponent } from './discovery-address-input.component';

describe('DiscoveryAddressInputComponent', () => {
  let component: DiscoveryAddressInputComponent;
  let fixture: ComponentFixture<DiscoveryAddressInputComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ DiscoveryAddressInputComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(DiscoveryAddressInputComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
