import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { EditableKeyComponent } from './editable-key.component';

describe('EditableKeyComponent', () => {
  let component: EditableKeyComponent;
  let fixture: ComponentFixture<EditableKeyComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ EditableKeyComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(EditableKeyComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
