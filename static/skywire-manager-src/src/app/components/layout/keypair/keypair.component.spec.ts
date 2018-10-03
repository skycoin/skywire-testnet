import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { KeypairComponent } from './keypair.component';

describe('KeypairComponent', () => {
  let component: KeypairComponent;
  let fixture: ComponentFixture<KeypairComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ KeypairComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(KeypairComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
