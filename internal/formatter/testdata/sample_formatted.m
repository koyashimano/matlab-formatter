function y = foo(x)

    if x > 0
        y = x + 1;
    elseif x < 0
        y = x -1;
    else
        y = 0;
    end

    switch x
        case 1
            disp('one');
        otherwise
            disp("other");
    end

    A = [1, 2, 3;
        4, 5, 6];
    B = {1, 2;
         3, 4};
    C = 1 + 2 * 3;
    D = 1.^2;
    E = 1 ...
        +2;
    F =- -1;
    G = a + b;
    H = 1/2;
    % formatter ignore 2
    ignoreThis =  5;
    stillIgnored =6;
    afterIgnore = 7;
%{
  block comment
%}
    if longStatement && ...
            anotherCondition
        disp('continued');
    end

end
