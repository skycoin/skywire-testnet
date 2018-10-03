import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { EditLabelComponent } from './edit-label.component';

describe('EditLabelComponent', () => {
  let component: EditLabelComponent;
  let fixture: ComponentFixture<EditLabelComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ EditLabelComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(EditLabelComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
