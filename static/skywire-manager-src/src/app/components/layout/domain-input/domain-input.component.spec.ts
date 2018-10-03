import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { DomainInputComponent } from './domain-input.component';

describe('DomainInputComponent', () => {
  let component: DomainInputComponent;
  let fixture: ComponentFixture<DomainInputComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ DomainInputComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(DomainInputComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
