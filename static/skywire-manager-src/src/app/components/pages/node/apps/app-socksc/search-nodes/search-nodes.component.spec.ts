import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SearchNodesComponent } from './search-nodes.component';

describe('SearchNodesComponent', () => {
  let component: SearchNodesComponent;
  let fixture: ComponentFixture<SearchNodesComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SearchNodesComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SearchNodesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
