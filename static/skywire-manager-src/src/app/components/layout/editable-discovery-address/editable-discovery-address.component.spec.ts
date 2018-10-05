import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { EditableDiscoveryAddressComponent } from './editable-discovery-address.component';

describe('EditableDiscoveryAddressComponent', () => {
  let component: EditableDiscoveryAddressComponent;
  let fixture: ComponentFixture<EditableDiscoveryAddressComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ EditableDiscoveryAddressComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(EditableDiscoveryAddressComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
